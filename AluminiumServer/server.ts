import express, { type Request, type Response, type NextFunction } from 'express';
import crypto from 'crypto';
import multer from 'multer';
import { MongoClient, GridFSBucket, ObjectId } from 'mongodb';
import { Readable } from 'stream';

const app = express();
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
  owner: ObjectId;
  uploadedAt: Date;
}

interface User {
  _id?: ObjectId;
  username: string;
  passwordHash: string;
  email: string;
  hashedTokens: string[];
  scopes: string[]; // e.g., ['read', 'write', 'admin']
  createdAt: Date;
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
    
app.post('/api/uploadPrebuilt', upload.single('package'), async (req: Request, res: Response, next: NextFunction): Promise<any> => {
  try {
    const { name, version } = req.body;
    const file = req.file;

    if (!name || !version || !file) {
      return res.status(400).json({ error: 'Missing package identifier metadata or binary file payload.' });
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
  try {
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

initDB().then(() => {
    app.listen(port, () => {
        console.log(`Server is running on http://localhost:${port}`);
    });
});