/**
 * XRU (XyPriss Rule Unit) Installer
 * Downloads the pre-built binary for your architecture from GitHub Releases.
 */

const http = require('https');
const fs = require('fs');
const path = require('path');

const REPO = 'Nehonix-Team/xru';
const BINARY_NAME = 'xru';

const colors = {
    reset: "\x1b[0m",
    green: "\x1b[32m",
    blue: "\x1b[34m",
    cyan: "\x1b[36m",
    red: "\x1b[31m"
};

async function getLatestVersion() {
    return new Promise((resolve, reject) => {
        const options = {
            hostname: 'api.github.com',
            path: `/repos/${REPO}/releases/latest`,
            headers: { 'User-Agent': 'node.js' }
        };
        http.get(options, (res) => {
            let data = '';
            res.on('data', (chunk) => data += chunk);
            res.on('end', () => {
                const json = JSON.parse(data);
                resolve(json.tag_name);
            });
        }).on('error', reject);
    });
}

function getPlatform() {
    const os = process.platform;
    const arch = process.arch;

    let goOS = os;
    if (os === 'win32') goOS = 'windows';
    
    let goArch = arch;
    if (arch === 'x64') goArch = 'amd64';

    return { os: goOS, arch: goArch };
}

async function download(url, dest) {
    return new Promise((resolve, reject) => {
        const file = fs.createWriteStream(dest);
        http.get(url, (res) => {
            if (res.statusCode === 302) { // Handle redirect
                return download(res.headers.location, dest).then(resolve).catch(reject);
            }
            res.pipe(file);
            file.on('finish', () => {
                file.close(resolve);
            });
        }).on('error', (err) => {
            fs.unlink(dest, () => {});
            reject(err);
        });
    });
}

async function install() {
    try {
        console.log(`${colors.cyan}[INFO]${colors.reset} Initializing XRU Installer...`);
        const version = await getLatestVersion();
        const { os, arch } = getPlatform();
        const ext = os === 'windows' ? '.exe' : '';
        const fileName = `${BINARY_NAME}-${os}-${arch}${ext}`;
        const url = `https://github.com/${REPO}/releases/latest/download/${fileName}`;

        const dest = path.join(process.cwd(), BINARY_NAME + ext);

        console.log(`${colors.blue}[INFO]${colors.reset} Fetching XRU ${version} for ${os}/${arch}...`);
        await download(url, dest);

        if (os !== 'windows') {
            fs.chmodSync(dest, '755');
        }

        console.log(`${colors.green}[SUCCESS]${colors.reset} XRU successfully installed at: ${dest}`);
        console.log(`Usage: ./${BINARY_NAME} version`);

    } catch (err) {
        console.error(`${colors.red}[ERROR]${colors.reset} Installation failed:`, err.message);
        process.exit(1);
    }
}

install();
