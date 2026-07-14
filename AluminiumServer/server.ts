import express, { type Request, type Response, type NextFunction } from 'express';
import crypto from 'crypto';
import bcrypt from 'bcrypt';
import multer from 'multer';
import { MongoClient, GridFSBucket, ObjectId } from 'mongodb';
import { Readable } from 'stream';
import {
  validatePackageName,
  validatePackageVersion,
  validateBuildFlags,
  validateSourceDir,
  validateCustomScript,
  isValidBuildSystem,
} from './buildValidation.js';

// peak server for aluminium

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
  buildSystem: 'cmake' | 'make' | 'meson' | 'custom' | 'none';
  dependencies?: string[];
  prebuiltBinaries?: string[];
  buildSetup?: BuildSetup;
  owner: ObjectId;
  uploadedAt: Date;
}

interface BuildSetup {
  buildScript: string;
  installScript: string;
  uninstallScript: string;
  sourceCodeUrl?: string;
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
  first8Chars: string; // For easier identification
  createdAt: Date;
  scopes: string[];
  owner: ObjectId;
}

const DB_NAME = process.env.DB_NAME || 'aluminium';

/**
 * Validates permission by evaluating both user and token capabilities.
 * Supports hierarchical fallback where 'admin' bypasses standard scope requirements.
 */
function hasScope(userScopes: string[], tokenScopes: string[], requiredScope: string): boolean {
  const hasUserScope = userScopes.includes(requiredScope) || userScopes.includes('admin');
  const hasTokenScope = tokenScopes.includes(requiredScope) || tokenScopes.includes('admin');
  return hasUserScope && hasTokenScope;
}

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
  const files = await bucket.find().limit(10).toArray();
  res.send(`\nWe have files: ${files.map(file => file.filename).join(", ")}`);
});

app.post('/api/createUser', async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const { username, password, email } = req.body;
  if (!username || !password || !email) {
    return res.status(400).json({ error: 'Missing required user fields.' });
  }

  try {
    const passwordHash = await bcrypt.hash(password, 10);
    const newUser: User = {
      username,
      passwordHash,
      email,
      tokens: [],
      scopes: ['read'], 
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
  if (!username || !password) {
    return res.status(400).json({ error: 'Missing required fields for token generation.' });
  }
  const scopesArray = Array.isArray(scopes)
    ? scopes.filter((scope): scope is string => typeof scope === 'string' && scope.trim().length > 0)
    : typeof scopes === 'string' && scopes.trim().length > 0
      ? [scopes.trim()]
      : [];

  try {
    const users = client.db(DB_NAME).collection<User>('users');
    const user = await users.findOne({ username });
    if (!user) {
      return res.status(404).json({ error: 'User not found.' });
    }

    const finalScopes = scopesArray.length > 0 ? scopesArray : user.scopes;

    for (const scope of finalScopes) {
      if (!['read', 'write', 'admin', 'dev'].includes(scope)) {
        return res.status(400).json({ error: `Invalid scope: ${scope}. Allowed scopes are 'read', 'write', 'admin', 'dev'.` });
      }
    }

    // Admins can mint tokens for any scope they desire
    const isAdmin = user.scopes.includes('admin');
    for (const scope of finalScopes) {
      if (!isAdmin && !user.scopes.includes(scope)) {
        return res.status(403).json({ error: `User does not have the required scope: ${scope}.` });
      }
    }

    const isMatch = await bcrypt.compare(password, user.passwordHash);
    if (!isMatch) {
      return res.status(401).json({ error: 'Invalid credentials.' });
    }

    const tokenValue = "ALUM+" + crypto.randomBytes(32).toString('hex');
    const hashedValue = crypto.createHash('sha256').update(tokenValue).digest('hex');

    const newToken: Token = {
      hashedValue,
      first8Chars: tokenValue.substring(0, 8), // Store first 8 chars for easier identification
      createdAt: new Date(),
      scopes: finalScopes,
      owner: user._id!,
    };

    await users.updateOne({ _id: user._id }, { $push: { tokens: newToken } });

    return res.status(201).json({ token: tokenValue, message: 'Token generated successfully.' });
  } catch (error) {
    next(error);
  }
});

app.get('/api/listTokens', async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const authHeader = req.headers['authorization'];
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'Malformed or missing authorization header.' });
  }
  const tokenValue = authHeader.substring(7);
  if (!tokenValue) {
    return res.status(400).json({ error: 'Missing authorization token.' });
  }
  try{
    const hashedValue = crypto.createHash('sha256').update(tokenValue).digest('hex');
    const users = client.db(DB_NAME).collection<User>('users');

    const user = await users.findOne({ 'tokens.hashedValue': hashedValue });
    if (!user) {
      return res.status(404).json({ error: 'Token not found or invalid.' });
    }

    return res.status(200).json({ tokens: user.tokens.map(t => ({ first8Chars: t.first8Chars, createdAt: t.createdAt, scopes: t.scopes })) });
  }
  catch (error) {
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

    const isMatch = await bcrypt.compare(password, user.passwordHash);
    if (!isMatch) {
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

    const activeToken = user.tokens.find(t => t.hashedValue === hashedValue);
    return res.status(200).json({ message: 'Token is valid.', userId: user._id, scopes: activeToken?.scopes });
  } catch (error) {
    next(error);
  }
});

app.post('/api/grantScope', async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const authHeader = req.headers['authorization'];
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'Malformed or missing authorization header.' });
  }
  const tokenValue = authHeader.substring(7);
  const { scope, username } = req.body;
  if (!tokenValue || !scope || !username) {
    return res.status(400).json({ error: 'Missing required fields for granting scope.' });
  }

  if (!['read', 'write', 'admin', 'dev'].includes(scope)) {
    return res.status(400).json({ error: `Invalid scope: ${scope}. Allowed scopes are 'read', 'write', 'admin', 'dev'.` });
  }
  
  try {
    const hashedValue = crypto.createHash('sha256').update(tokenValue).digest('hex');
    const users = client.db(DB_NAME).collection<User>('users');
    const requestingUser = await users.findOne({ 'tokens.hashedValue': hashedValue });

    if(!requestingUser) {
      return res.status(404).json({ error: 'Token not found or invalid.' });
    }

    const token = requestingUser.tokens.find(t => t.hashedValue === hashedValue);
    if (!token || !hasScope(requestingUser.scopes, token.scopes, 'admin')) {
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
  const authHeader = req.headers['authorization'];
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'Malformed or missing authorization header.' });
  }
  const tokenValue = authHeader.substring(7);
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

    return res.status(200).json({ scopes: user.scopes });
  } catch (error) {
    next(error);
  }
});

app.get('/api/getTokenScopes', async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const authHeader = req.headers['authorization'];
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'Malformed or missing authorization header.' });
  }
  const tokenValue = authHeader.substring(7);
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

app.post('/api/uploadPrebuilt', async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const authHeader = req.headers['authorization'];
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'Malformed or missing authorization header.' });
  }
  const tokenValue = authHeader.substring(7);
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

    // 1. Verify general write permissions (Safe to do early)
    if (!hasScope(user.scopes, token.scopes, 'write')) {
      return res.status(403).json({ error: 'Insufficient permissions to upload prebuilt binaries.' });
    }

    // 2. Determine file size limit based on scope hierarchy
    const isPrivileged = user.scopes.includes('admin') || user.scopes.includes('dev') ||
                        token.scopes.includes('admin') || token.scopes.includes('dev');
                        
    const MAX_FILE_SIZE = isPrivileged ? 2 * 1024 * 1024 * 1024 : 50 * 1024 * 1024; 

    // 3. Initialize dynamic multer instance
    const dynamicUpload = multer({
      storage: multer.memoryStorage(),
      limits: { fileSize: MAX_FILE_SIZE }
    }).single('package');

    // 4. Run the upload handler manually
    dynamicUpload(req, res, async (err) => {
      if (err instanceof multer.MulterError) {
        if (err.code === 'LIMIT_FILE_SIZE') {
          return res.status(400).json({ 
            error: `File size limit exceeded. Non-admin threshold is ${MAX_FILE_SIZE / (1024 * 1024)}MB.` 
          });
        }
        return res.status(400).json({ error: err.message });
      } else if (err) {
        return next(err);
      }

      // --- CRITICAL CHANGE: Move body-dependent logic INSIDE the callback ---
      const { name, version, packageName } = req.body;
      const file = req.file;

      if (!name || !version || !file || !packageName) {
        return res.status(400).json({ error: 'Missing package identifier metadata or binary file payload.' });
      }

      const existingPackage = await client.db(DB_NAME).collection<Package>('packages').findOne({
        name: packageName,
        version
      });

      if (!existingPackage) {
        return res.status(404).json({ error: 'No package matches the provided packageName.' });
      }

      const packageUser = await users.findOne({ _id: existingPackage.owner });
      if (!packageUser) {
        return res.status(404).json({ error: 'Package owner not found.' });
      }

      if (!packageUser._id.equals(user._id) && !user.scopes.includes('admin')) {
        return res.status(403).json({ error: 'You do not have permission to upload binaries for this package.' });
      }

      // GridFS upload logic stays here...
      const metadata = { name, version };
      const bucketFilename = `${name}@${version}.tar.gz`;
      const uploadStream = bucket.openUploadStream(bucketFilename, { metadata });

      const readableFileStream = new Readable();
      readableFileStream.push(file.buffer);
      readableFileStream.push(null);

      readableFileStream.pipe(uploadStream)
        .on('error', (uploadErr) => {
          console.error('GridFS Upload Error:', uploadErr);
          return res.status(500).json({ error: 'Stream interrupted during database persist operations.' });
        })
        .on('finish', () => {
          return res.status(201).json({ 
            message: `Package ${name}@${version} uploaded successfully!`,
            fileId: uploadStream.id
          });
        });
    });
  } catch (error) {
    next(error);
  }
});

app.post('/api/downloadPrebuilt', async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const authHeader = req.headers['authorization'];
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'Malformed or missing authorization header.' });
  }
  const tokenValue = authHeader.substring(7);
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
    if (!token || !hasScope(user.scopes, token.scopes, 'read')) {
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
  const authHeader = req.headers['authorization'];
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'Malformed or missing authorization header.' });
  }
  const tokenValue = authHeader.substring(7);
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
    if (!token || !hasScope(user.scopes, token.scopes, 'write')) {
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
    const packageDoc = await packagesCollection.findOne({ name, version });

    if (!packageDoc) {
      return res.status(404).json({ error: 'Associated package document not found.' });
    }

    if (!packageDoc.owner.equals(user._id) && !user.scopes.includes('admin')) {
      return res.status(403).json({ error: 'You do not have permission to delete binaries for this package.' });
    }

    await bucket.delete(matchedPackage._id);
    return res.status(200).json({ message: `Prebuilt binary ${name}@${version} deleted successfully.` });
  } catch (error) {
    next(error);
  }
});

app.post('/api/deletePackage', async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const authHeader = req.headers['authorization'];
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'Malformed or missing authorization header.' });
  }
  const tokenValue = authHeader.substring(7);
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
    if (!token || !hasScope(user.scopes, token.scopes, 'write')) {
      return res.status(403).json({ error: 'Insufficient permissions to delete packages.' });
    }

    const { name, version } = req.body;
    if (!name || !version) {
      return res.status(400).json({ error: 'Missing package identifier metadata.' });
    }

    const packagesCollection = client.db(DB_NAME).collection<Package>('packages');
    const packageDoc = await packagesCollection.findOne({ name, version });

    if (!packageDoc) {
      return res.status(404).json({ error: 'Package not found.' });
    }

    if (!packageDoc.owner.equals(user._id) && !user.scopes.includes('admin')) {
      return res.status(403).json({ error: 'You do not have permission to delete this package.' });
    }

    await packagesCollection.deleteOne({ _id: packageDoc._id });
    return res.status(200).json({ message: `Package ${name}@${version} deleted successfully.` });
  } catch (error) {
    next(error);
  }
});

app.post('/api/registerPackage', async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const authHeader = req.headers['authorization'];
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'Malformed or missing authorization header.' });
  }
  const tokenValue = authHeader.substring(7);
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
    if (!token || !hasScope(user.scopes, token.scopes, 'write')) {
      return res.status(403).json({ error: 'Insufficient permissions to register packages.' });
    }

    const { name, version, buildSystem, dependencies } = req.body;

    if (!validatePackageName(name) || !validatePackageVersion(version)) {
      return res.status(400).json({ error: 'Invalid package name or version.' });
    }

    if (!isValidBuildSystem(buildSystem)) {
      return res.status(400).json({ error: 'Invalid or missing build system specified.' });
    }

    if (buildSystem === 'custom' && !hasScope(user.scopes, token.scopes, 'dev')) {
      return res.status(403).json({ error: 'Insufficient permissions to register custom build systems.' });
    }

    if (dependencies !== undefined && !Array.isArray(dependencies)) {
      return res.status(400).json({ error: 'Dependencies must be an array when provided.' });
    }

    const safeName = name;
    const packagePayload: Package = {
      name: safeName,
      version,
      buildSystem,
      dependencies,
      owner: user._id!,
      uploadedAt: new Date()
    };

    if (buildSystem === 'custom') {
      const { customBuildScript, customInstallScript, customUninstallScript } = req.body;
      if (!validateCustomScript(customBuildScript) || !validateCustomScript(customInstallScript) || !validateCustomScript(customUninstallScript)) {
        return res.status(400).json({ error: 'Custom build scripts contain unsafe shell characters or are malformed.' });
      }
      packagePayload.buildSetup = {
        buildScript: customBuildScript,
        installScript: customInstallScript,
        uninstallScript: customUninstallScript
      };
    } else if (buildSystem === 'none') {
      packagePayload.buildSystem = buildSystem;
    } else if (buildSystem === 'cmake') {
      const { buildFlags, sourceDir } = req.body;
      if (!validateBuildFlags(buildFlags)) {
        return res.status(400).json({ error: 'Invalid build flags specified.' });
      }
      if (!validateSourceDir(sourceDir)) {
        return res.status(400).json({ error: 'Invalid source directory specified.' });
      }
      const buildFlagsSafe = typeof buildFlags === 'string' ? buildFlags.trim() : '';
      packagePayload.buildSetup = {
        sourceCodeUrl: typeof sourceDir === 'string' ? sourceDir : '',
        buildScript: `cmake -B build ${buildFlagsSafe} -DCMAKE_INSTALL_PREFIX="$HOME/.aluminium/install/${safeName}" -S . && cmake --build build`,
        installScript: `cmake --install build && cp build/install_manifest.txt "$HOME/.aluminium/install/${safeName}/install_manifest.txt"`,
        uninstallScript: `xargs rm -f < "$HOME/.aluminium/install/${safeName}/install_manifest.txt" && rm -rf "$HOME/.aluminium/install/${safeName}"`,
      };
    } else if (buildSystem === 'make') {
      const { buildFlags, sourceDir } = req.body;
      if (!validateBuildFlags(buildFlags)) {
        return res.status(400).json({ error: 'Invalid build flags specified.' });
      }
      if (!validateSourceDir(sourceDir)) {
        return res.status(400).json({ error: 'Invalid source directory specified.' });
      }
      const buildFlagsSafe = typeof buildFlags === 'string' ? buildFlags.trim() : '';
      packagePayload.buildSetup = {
        sourceCodeUrl: typeof sourceDir === 'string' ? sourceDir : '',
        buildScript: `mkdir build && (../configure ${buildFlagsSafe} --prefix="$HOME/.aluminium/install/${safeName}" || ../Configure ${buildFlagsSafe} --prefix="$HOME/.aluminium/install/${safeName}")  && make`,
        installScript: `make install`,
        uninstallScript: `rm -rf "$HOME/.aluminium/install/${safeName}"`,
      };
    } else if (buildSystem === 'meson') {
      const { buildFlags, sourceDir } = req.body;
      if (!validateBuildFlags(buildFlags)) {
        return res.status(400).json({ error: 'Invalid build flags specified.' });
      }
      if (!validateSourceDir(sourceDir)) {
        return res.status(400).json({ error: 'Invalid source directory specified.' });
      }
      const buildFlagsSafe = typeof buildFlags === 'string' ? buildFlags.trim() : '';
      packagePayload.buildSetup = {
        sourceCodeUrl: typeof sourceDir === 'string' ? sourceDir : '',
        buildScript: `meson setup build ${buildFlagsSafe} --prefix="$HOME/.aluminium/install/${safeName}" && meson compile -C build`,
        installScript: `meson install -C build`,
        uninstallScript: `rm -rf "$HOME/.aluminium/install/${safeName}"`,
      };
    }


    const collection = client.db(DB_NAME).collection<Package>('packages');
    await collection.insertOne(packagePayload);

    return res.status(201).json({ message: 'Package registered successfully.' });
  } catch (error) {
    next(error);
  }
});

app.get('/api/listPackages', async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const authHeader = req.headers['authorization'];
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'Malformed or missing authorization header.' });
  }
  const tokenValue = authHeader.substring(7);
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
    if (!token || !hasScope(user.scopes, token.scopes, 'read')) {
      return res.status(403).json({ error: 'Insufficient permissions to list packages.' });
    }

    const collection = client.db(DB_NAME).collection<Package>('packages');
    const packages = await collection.find().toArray();

    return res.status(200).json({ packages });
  } catch (error) {
    next(error);
  }
});

app.get('/api/getPackage', async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  const authHeader = req.headers['authorization'];
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'Malformed or missing authorization header.' });
  }
  const tokenValue = authHeader.substring(7);
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
    if (!token || !hasScope(user.scopes, token.scopes, 'read')) {
      return res.status(403).json({ error: 'Insufficient permissions to retrieve package details.' });
    }

    const { name, version } = req.body;
    if (!name || !version) {
      return res.status(400).json({ error: 'Missing package identifier metadata.' });
    }

    const collection = client.db(DB_NAME).collection<Package>('packages');
    const packageDoc = await collection.findOne({ name: String(name), version: String(version) });

    if (!packageDoc) {
      return res.status(404).json({ error: 'Package not found.' });
    }

    return res.status(200).json({ package: packageDoc });
  } catch (error) {
    next(error);
  }
});

await initDB();
app.listen(port, () => {
    console.log(`Server is running on http://localhost:${port}`);
});