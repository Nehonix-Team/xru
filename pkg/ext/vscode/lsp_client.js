const cp = require('child_process');

class XruLspClient {
    constructor(binaryPath) {
        this.binaryPath = binaryPath;
        this.process = null;
        this.reqId = 1;
        this.pendingRequests = new Map();
        this.diagnosticListeners = [];
        this.exitListeners = [];
        this.buffer = Buffer.alloc(0);
    }

    onExit(cb) {
        this.exitListeners.push(cb);
    }

    start() {
        return new Promise((resolve, reject) => {
            try {
                this.process = cp.spawn(this.binaryPath, ['lsp'], {
                    stdio: ['pipe', 'pipe', 'pipe']
                });
            } catch (err) {
                return reject(err);
            }

            this.process.on('error', (err) => {
                this.cleanup();
            });

            this.process.on('exit', (code) => {
                this.cleanup();
                for (const l of this.exitListeners) {
                    try { l(code); } catch (e) {}
                }
            });

            this.process.stderr.on('data', (data) => {
                // stderr can be used for debug logs if needed
            });

            this.process.stdout.on('data', (chunk) => {
                this.handleData(chunk);
            });

            // Send initialize request
            this.request('initialize', {
                processId: process.pid,
                rootUri: null,
                capabilities: {}
            }).then((res) => {
                this.notify('initialized', {});
                resolve(res);
            }).catch(reject);
        });
    }

    isAlive() {
        return !!this.process && !this.process.killed;
    }

    onDiagnostics(callback) {
        this.diagnosticListeners.push(callback);
    }

    notify(method, params) {
        if (!this.isAlive()) return;
        const payload = JSON.stringify({
            jsonrpc: '2.0',
            method: method,
            params: params
        });
        const header = `Content-Length: ${Buffer.byteLength(payload, 'utf8')}\r\n\r\n`;
        this.process.stdin.write(header + payload);
    }

    request(method, params) {
        if (!this.isAlive()) {
            return Promise.reject(new Error('LSP server is not running'));
        }
        const id = this.reqId++;
        const payload = JSON.stringify({
            jsonrpc: '2.0',
            id: id,
            method: method,
            params: params
        });
        const header = `Content-Length: ${Buffer.byteLength(payload, 'utf8')}\r\n\r\n`;

        return new Promise((resolve, reject) => {
            this.pendingRequests.set(id, { resolve, reject });
            this.process.stdin.write(header + payload);
        });
    }

    handleData(chunk) {
        this.buffer = Buffer.concat([this.buffer, chunk]);

        while (true) {
            const headerEnd = this.buffer.indexOf('\r\n\r\n');
            if (headerEnd === -1) {
                break;
            }

            const headerStr = this.buffer.subarray(0, headerEnd).toString('utf8');
            const match = headerStr.match(/Content-Length:\s*(\d+)/i);
            if (!match) {
                // Malformed header, skip to after \r\n\r\n
                this.buffer = this.buffer.subarray(headerEnd + 4);
                continue;
            }

            const contentLength = parseInt(match[1], 10);
            const totalMsgLen = headerEnd + 4 + contentLength;

            if (this.buffer.length < totalMsgLen) {
                // Wait for more data
                break;
            }

            const bodyBytes = this.buffer.subarray(headerEnd + 4, totalMsgLen);
            this.buffer = this.buffer.subarray(totalMsgLen);

            try {
                const msg = JSON.parse(bodyBytes.toString('utf8'));
                this.handleMessage(msg);
            } catch (e) {
                // JSON parse error
            }
        }
    }

    handleMessage(msg) {
        // Notification
        if (msg.method === 'textDocument/publishDiagnostics') {
            for (const listener of this.diagnosticListeners) {
                try {
                    listener(msg.params);
                } catch (e) {
                    // listener error
                }
            }
            return;
        }

        // Response
        if (msg.id !== undefined && this.pendingRequests.has(msg.id)) {
            const { resolve, reject } = this.pendingRequests.get(msg.id);
            this.pendingRequests.delete(msg.id);
            if (msg.error) {
                reject(new Error(msg.error.message || 'LSP request failed'));
            } else {
                resolve(msg.result);
            }
        }
    }

    stop() {
        if (this.isAlive()) {
            this.notify('exit', {});
            try {
                this.process.kill();
            } catch (e) {}
        }
        this.cleanup();
    }

    cleanup() {
        this.process = null;
        for (const [id, req] of this.pendingRequests.entries()) {
            req.reject(new Error('LSP process exited'));
        }
        this.pendingRequests.clear();
    }
}

module.exports = { XruLspClient };
