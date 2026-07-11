import express, { type Request, type Response, type NextFunction } from 'express';
import crypto from 'crypto';
import multer from 'multer';
import { MongoClient, GridFSBucket, ObjectId } from 'mongodb';
import { Readable } from 'stream';

const app = express();
app.use(express.json());
const port = process.env.PORT || 3000;

const storage = multer.memoryStorage();
const upload = multer({ storage });

let bucket: GridFSBucket;
let client: MongoClient;

interface Package {
  _id?: ObjectId;
  name: string;
  version: string;
  buildSystem?: string;
  dependacies?: string[];
  prebuiltBinaries?: string[];
  customBuildSystem?: CustomBuildSystem;
  owner: ObjectId;
  uploadedAt: Date;
}

interface CustomBuildSystem {
  customBuildScript: string;
  customInstallScript: string;
  customUninstallScript: string;
}

interface User {
  _id?: ObjectId;
  username: string;
  passwordHash: string;
  email: string;
  tokens: Token[];
  scopes: string[]; // e.g., ['read', 'write', 'admin', 'dev']
  createdAt: Date;
}

interface Token {
  hashedValue: string;
  createdAt: Date;
  scopes: string[];
  owner: ObjectId;
}

const DB_NAME = process.env.DB_NAME || 'aluminium';

async function initDB() {
    try {
        client = new MongoClient(process.env.MONGO_URI || 'mongodb://localhost:27017');
        await client.connect();
        const db = client.db(DB_NAME);
        bucket = new GridFSBucket(db, { bucketName: 'packages' });
        await db.collection('packages.files').createIndex({ 
            'metadata.name': 1, 
            'metadata.version': 1,
        });
        console.log('Compound metadata index established.');
    } catch (error) {
        console.error('Error initializing database:', error);
        process.exit(1);
    }
}

app.get('/', async (req, res) => {
  //res.send('Hello, Aluminium Server!');
  res.send(`\nWe have files: ${await bucket.find().toArray().then(files => files.map(file => file.filename).join(", "))}`);
});

app.post('/api/createUser', async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const { username, password, email } = req.body;
  if (!username || !password || !email) {
    return res.status(400).json({ error: 'Missing required user fields.' });
  }

  try {
    const passwordHash = crypto.createHash('sha256').update(password).digest('hex');
    const newUser: User = {
      username,
      passwordHash,
      email,
      tokens: [],
      scopes: ['read'], // Default scopes
      createdAt: new Date(),
    };
    const users = client.db(DB_NAME).collection<User>('users');
    await users.insertOne(newUser);
    return res.status(201).json({ message: 'User created successfully.' });
  } catch (error) {
    next(error);
  }
});

app.post('/api/generateToken', async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const { username, password, scopes } = req.body;
  if (!username || !password || !scopes) {
    return res.status(400).json({ error: 'Missing required fields for token generation.' });
  }
  const scopesArray = Array.isArray(scopes) ? scopes : [scopes];

  for (const scope of scopesArray) {
    if (!['read', 'write', 'admin', 'dev'].includes(scope)) {
      return res.status(400).json({ error: `Invalid scope: ${scope}. Allowed scopes are 'read', 'write', 'admin'.` });
    }
  }

  try {
    const users = client.db(DB_NAME).collection<User>('users');
    const user = await users.findOne({ username });
    if (!user) {
      return res.status(404).json({ error: 'User not found.' });
    }
    for (const scope of scopesArray) {
      if (user.scopes.indexOf(scope) === -1) {
        return res.status(403).json({ error: `User does not have the required scope: ${scope}.` });
      }
    }

    const passwordHash = crypto.createHash('sha256').update(password).digest('hex');
    if (user.passwordHash !== passwordHash) {
      return res.status(401).json({ error: 'Invalid credentials.' });
    }

    const tokenValue = "ALUM+" + crypto.randomBytes(32).toString('hex');
    const hashedValue = crypto.createHash('sha256').update(tokenValue).digest('hex');

    const newToken: Token = {
      hashedValue,
      createdAt: new Date(),
      scopes: scopesArray,
      owner: user._id!,
    };

    await users.updateOne({ _id: user._id }, { $push: { tokens: newToken } });

    return res.status(201).json({ token: tokenValue, message: 'Token generated successfully.' });
  } catch (error) {
    next(error);
  }
});

app.post('/api/revokeToken', async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const { token, password } = req.body;
  if (!token || !password) {
    return res.status(400).json({ error: 'Missing required fields for token revocation.' });
  }

  try {
    const hashedValue = crypto.createHash('sha256').update(token).digest('hex');
    const users = client.db(DB_NAME).collection<User>('users');
    const user = await users.findOne({ 'tokens.hashedValue': hashedValue });

    if (!user) {
      return res.status(404).json({ error: 'Token not found or invalid.' });
    }

    const passwordHash = crypto.createHash('sha256').update(password).digest('hex');
    if (user.passwordHash !== passwordHash) {
      return res.status(401).json({ error: 'Invalid credentials.' });
    }

    await users.updateOne(
      { _id: user._id },
      { $pull: { tokens: { hashedValue } } }
    );

    return res.status(200).json({ message: 'Token revoked successfully.' });
  } catch (error) {
    next(error);
  }
});

app.post('/api/validateToken', async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const { token } = req.body;
  if (!token) {
    return res.status(400).json({ error: 'Missing token for validation.' });
  }

  try {
    const hashedValue = crypto.createHash('sha256').update(token).digest('hex');
    const users = client.db(DB_NAME).collection<User>('users');
    const user = await users.findOne({ 'tokens.hashedValue': hashedValue });

    if (!user) {
      return res.status(404).json({ error: 'Token not found or invalid.' });
    }

    return res.status(200).json({ message: 'Token is valid.', userId: user._id, scopes: user.tokens.find(t => t.hashedValue === hashedValue)?.scopes });
  } catch (error) {
    next(error);
  }
});

app.post('/api/grantScope', async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const tokenValue = req.headers['authorization']?.split(' ')[1];
  const { scope, username } = req.body;
  if (!tokenValue || !scope || !username) {
    return res.status(400).json({ error: 'Missing required fields for granting scope.' });
  }

  if (!['read', 'write', 'admin'].includes(scope)) {
    return res.status(400).json({ error: `Invalid scope: ${scope}. Allowed scopes are 'read', 'write', 'admin'.` });
  }
  
  try {
    const hashedValue = crypto.createHash('sha256').update(tokenValue).digest('hex');
    const users = client.db(DB_NAME).collection<User>('users');
    const requestingUser = await users.findOne({ 'tokens.hashedValue': hashedValue });

    if(!requestingUser) {
      return res.status(404).json({ error: 'Token not found or invalid.' });
    }

    if (!requestingUser.scopes.includes('admin')) {
      return res.status(403).json({ error: 'Insufficient permissions to grant scopes.' });
    }

    const targetUser = await users.findOne({ username });
    if (!targetUser) {
      return res.status(404).json({ error: 'Target user not found.' });
    }

    if (targetUser.scopes.includes(scope)) {
      return res.status(400).json({ error: `User already has the scope: ${scope}.` });
    }

    await users.updateOne(
      { _id: targetUser._id },
      { $push: { scopes: scope } }
    );

    return res.status(200).json({ message: `Scope '${scope}' granted to user '${username}' successfully.` });
  } catch (error) {
    next(error);
  }
});

app.get('/api/getUserScopes', async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const tokenValue = req.headers['authorization']?.split(' ')[1];
  if (!tokenValue) {
    return res.status(400).json({ error: 'Missing authorization token.' });
  }

  try {
    const hashedValue = crypto.createHash('sha256').update(tokenValue).digest('hex');
    const users = client.db(DB_NAME).collection<User>('users');
    const user = await users.findOne({ 'tokens.hashedValue': hashedValue });

    if (!user) {
      return res.status(404).json({ error: 'Token not found or invalid.' });
    }

    const token = user.tokens.find(t => t.hashedValue === hashedValue);
    if (!token) {
      return res.status(404).json({ error: 'Token not found.' });
    }

    return res.status(200).json({ scopes: token.scopes });
  } catch (error) {
    next(error);
  }
});

app.get('/api/getTokenScopes', async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const tokenValue = req.headers['authorization']?.split(' ')[1];
  if (!tokenValue) {
    return res.status(400).json({ error: 'Missing authorization token.' });
  }

  try {
    const hashedValue = crypto.createHash('sha256').update(tokenValue).digest('hex');
    const users = client.db(DB_NAME).collection<User>('users');
    const user = await users.findOne({ 'tokens.hashedValue': hashedValue });

    if (!user) {
      return res.status(404).json({ error: 'Token not found or invalid.' });
    }

    const token = user.tokens.find(t => t.hashedValue === hashedValue);
    if (!token) {
      return res.status(404).json({ error: 'Token not found.' });
    }

    return res.status(200).json({ scopes: token.scopes });
  } catch (error) {
    next(error);
  }
});

app.post('/api/uploadPrebuilt', upload.single('package'), async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const tokenValue = req.headers['authorization']?.split(' ')[1];
  if (!tokenValue) {
    return res.status(400).json({ error: 'Missing authorization token.' });
  }
  
  try {
    const hashedValue = crypto.createHash('sha256').update(tokenValue).digest('hex');
    const users = client.db(DB_NAME).collection<User>('users');
    const user = await users.findOne({ 'tokens.hashedValue': hashedValue });

    if (!user) {
      return res.status(404).json({ error: 'Token not found or invalid.' });
    }
    if(user.scopes.indexOf('write') === -1 || user.tokens.find(t => t.hashedValue === hashedValue)?.scopes.indexOf('write') === -1) {
      return res.status(403).json({ error: 'Insufficient permissions to upload prebuilt binaries.' });
    }
    const { name, version, packageName } = req.body;
    const file = req.file;

    if (!name || !version || !file || !packageName) {
      return res.status(400).json({ error: 'Missing package identifier metadata or binary file payload.' });
    }

    const existingPackage = await client.db(DB_NAME).collection<Package>('packages').findOne({
      packageName,
      version
    });

    if (!existingPackage) {
      return res.status(404).json({ error: 'No package matches the provided packageName.' });
    }

    const packageUser = await users.findOne({ _id: existingPackage.owner });
    if (!packageUser) {
      return res.status(404).json({ error: 'Package owner not found.' });
    }
    if (!packageUser._id.equals(user._id) && user.scopes.indexOf('admin') === -1) {
      return res.status(403).json({ error: 'You do not have permission to upload binaries for this package.' });
    }

    const metadata = { name, version };
    const bucketFilename = `${name}@${version}.tar.gz`;
    const uploadStream = bucket.openUploadStream(bucketFilename, { metadata });

    const readableFileStream = new Readable();
    readableFileStream.push(file.buffer);
    readableFileStream.push(null);

    readableFileStream.pipe(uploadStream)
      .on('error', (err) => {
        console.error('GridFS Upload Error:', err);
        return res.status(500).json({ error: 'Stream interrupted during database persist operations.' });
      })
      .on('finish', () => {
        return res.status(201).json({ 
          message: `Package ${name}@${version} for uploaded successfully!`,
          fileId: uploadStream.id
        });
      });
  } catch (error) {
    next(error);
  }
});

app.post('/api/downloadPrebuilt', async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const tokenValue = req.headers['authorization']?.split(' ')[1];
  if (!tokenValue) {
    return res.status(400).json({ error: 'Missing authorization token.' });
  }

  try {
    const hashedValue = crypto.createHash('sha256').update(tokenValue).digest('hex');
    const users = client.db(DB_NAME).collection<User>('users');
    const user = await users.findOne({ 'tokens.hashedValue': hashedValue });

    if (!user) {
      return res.status(404).json({ error: 'Token not found or invalid.' });
    }
    if(user.scopes.indexOf('read') === -1 || user.tokens.find(t => t.hashedValue === hashedValue)?.scopes.indexOf('read') === -1) {
      return res.status(403).json({ error: 'Insufficient permissions to download prebuilt binaries.' });
    }
    const { name, version } = req.body;
    if (!name || !version) {
      return res.status(400).json({ error: 'Missing package identifier metadata.' });
    }

    const filesCollection = client.db(DB_NAME).collection('packages.files');
    const matchedPackage = await filesCollection.findOne({
      'metadata.name': name,
      'metadata.version': version
    });
    if (!matchedPackage) {
      return res.status(404).json({ error: 'No prebuilt binary matches your execution environment configuration.' });
    }

    res.setHeader('Content-Type', 'application/gzip');
    res.setHeader('Content-Disposition', `attachment; filename="${matchedPackage.filename}"`);

    const downloadStream = bucket.openDownloadStream(matchedPackage._id);

    downloadStream.pipe(res)
      .on('error', (err) => {
        console.error('GridFS Download Error:', err);
        if (!res.headersSent) {
          res.status(500).json({ error: 'Streaming binary pipeline crashed prematurely.' });
        }
      })
      .on('end', () => {
        res.end();
      });
  } catch (error) {
    next(error);
  }
});

app.post('/api/deletePrebuilt', async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const tokenValue = req.headers['authorization']?.split(' ')[1];
  if (!tokenValue) {
    return res.status(400).json({ error: 'Missing authorization token.' });
  }

  try {
    const hashedValue = crypto.createHash('sha256').update(tokenValue).digest('hex');
    const users = client.db(DB_NAME).collection<User>('users');
    const user = await users.findOne({ 'tokens.hashedValue': hashedValue });

    if (!user) {
      return res.status(404).json({ error: 'Token not found or invalid.' });
    }
    if(user.scopes.indexOf('write') === -1 || user.tokens.find(t => t.hashedValue === hashedValue)?.scopes.indexOf('write') === -1) {
      return res.status(403).json({ error: 'Insufficient permissions to delete prebuilt binaries.' });
    }
    const { name, version } = req.body;
    if (!name || !version) {
      return res.status(400).json({ error: 'Missing package identifier metadata.' });
    }

    const filesCollection = client.db(DB_NAME).collection('packages.files');
    const matchedPackage = await filesCollection.findOne({
      'metadata.name': name,
      'metadata.version': version
    });
    if (!matchedPackage) {
      return res.status(404).json({ error: 'No prebuilt binary matches your execution environment configuration.' });
    }

    const packagesCollection = client.db(DB_NAME).collection<Package>('packages');
    const packageDoc = await packagesCollection.findOne({
      name,
      version
    });

    if (!packageDoc) {
      return res.status(404).json({ error: 'Associated package document not found.' });
    }

    if (!packageDoc.owner.equals(user._id) && user.scopes.indexOf('admin') === -1) {
      return res.status(403).json({ error: 'You do not have permission to delete binaries for this package.' });
    }

    await bucket.delete(matchedPackage._id);
    return res.status(200).json({ message: `Prebuilt binary ${name}@${version} deleted successfully.` });
  } catch (error) {
    next(error);
  }
});

app.post('/api/deletePackage', async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const tokenValue = req.headers['authorization']?.split(' ')[1];
  if (!tokenValue) {
    return res.status(400).json({ error: 'Missing authorization token.' });
  }

  try {
    const hashedValue = crypto.createHash('sha256').update(tokenValue).digest('hex');
    const users = client.db(DB_NAME).collection<User>('users');
    const user = await users.findOne({ 'tokens.hashedValue': hashedValue });

    if (!user) {
      return res.status(404).json({ error: 'Token not found or invalid.' });
    }
    if(user.scopes.indexOf('write') === -1 || user.tokens.find(t => t.hashedValue === hashedValue)?.scopes.indexOf('write') === -1) {
      return res.status(403).json({ error: 'Insufficient permissions to delete packages.' });
    }

    const { name, version } = req.body;
    if (!name || !version) {
      return res.status(400).json({ error: 'Missing package identifier metadata.' });
    }

    const packagesCollection = client.db(DB_NAME).collection<Package>('packages');
    const packageDoc = await packagesCollection.findOne({
      name,
      version
    });

    if (!packageDoc) {
      return res.status(404).json({ error: 'Package not found.' });
    }

    if (!packageDoc.owner.equals(user._id) && user.scopes.indexOf('admin') === -1) {
      return res.status(403).json({ error: 'You do not have permission to delete this package.' });
    }

    await packagesCollection.deleteOne({ _id: packageDoc._id });

    return res.status(200).json({ message: `Package ${name}@${version} deleted successfully.` });
  } catch (error) {
    next(error);
  }
});

app.post('/api/registerPackage', async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const tokenValue = req.headers['authorization']?.split(' ')[1];
  if (!tokenValue) {
    return res.status(400).json({ error: 'Missing authorization token.' });
  }

  try {
    const hashedValue = crypto.createHash('sha256').update(tokenValue).digest('hex');
    const users = client.db(DB_NAME).collection<User>('users');
    const user = await users.findOne({ 'tokens.hashedValue': hashedValue });

    if (!user) {
      return res.status(404).json({ error: 'Token not found or invalid.' });
    }
    if(user.scopes.indexOf('write') === -1 || user.tokens.find(t => t.hashedValue === hashedValue)?.scopes.indexOf('write') === -1) {
      return res.status(403).json({ error: 'Insufficient permissions to register packages.' });
    }

    const { name, version, buildSystem, dependacies } = req.body;

    if(buildSystem && !['cmake', 'make', 'bazel', 'meson', 'custom'].includes(buildSystem)) {
      return res.status(400).json({ error: 'Invalid build system specified.' });
    }

    if(buildSystem === 'custom' && user.scopes.indexOf('dev') === -1) {
      return res.status(403).json({ error: 'Insufficient permissions to register custom build systems.' });
    }

    if(buildSystem === 'custom') {
      const { customBuildScript, customInstallScript, customUninstallScript } = req.body;
      if (!customBuildScript || !customInstallScript || !customUninstallScript) {
        return res.status(400).json({ error: 'Missing required fields for custom build system.' });
      }
      const customBuildSystem: CustomBuildSystem = {
        customBuildScript,
        customInstallScript,
        customUninstallScript
      };
      const collection = client.db(DB_NAME).collection<Package>('packages');
      collection.insertOne({
        name,
        version,
        buildSystem,
        dependacies,
        owner: user._id!,
        uploadedAt: new Date(),
        customBuildSystem
      })
    }else{
      const collection = client.db(DB_NAME).collection<Package>('packages');
      collection.insertOne({
        name,
        version,
        buildSystem,
        dependacies,
        owner: user._id!,
        uploadedAt: new Date()
      })
    }

    return res.status(201).json({ message: 'Package registered successfully.' });
  } catch (error) {
    next(error);
  }
});

    

initDB().then(() => {
    app.listen(port, () => {
        console.log(`Server is running on http://localhost:${port}`);
    });
});