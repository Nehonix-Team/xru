const vscode = require('vscode');
const cp = require('child_process');
const path = require('path');

let diagnosticCollection;

function activate(context) {
    diagnosticCollection = vscode.languages.createDiagnosticCollection('xru');
    context.subscriptions.push(diagnosticCollection);

    context.subscriptions.push(vscode.workspace.onDidSaveTextDocument(doc => {
        if (doc.languageId === 'xru') {
            updateDiagnostics(doc);
        }
    }));

    context.subscriptions.push(vscode.workspace.onDidOpenTextDocument(doc => {
        if (doc.languageId === 'xru') {
            updateDiagnostics(doc);
        }
    }));

    // Initial check
    if (vscode.window.activeTextEditor && vscode.window.activeTextEditor.document.languageId === 'xru') {
        updateDiagnostics(vscode.window.activeTextEditor.document);
    }
}

function updateDiagnostics(document) {
    const filePath = document.uri.fsPath;
    const workspaceFolder = vscode.workspace.getWorkspaceFolder(document.uri);
    const cwd = workspaceFolder ? workspaceFolder.uri.fsPath : path.dirname(filePath);

    // Tentative de localisation du binaire xru
    // 1. Dans le root du workspace
    // 2. Dans le PATH
    const xruLocal = path.join(cwd, 'xru');
    const cmd = `"${xruLocal}" "${filePath}"`;

    cp.exec(cmd, { cwd }, (err, stdout, stderr) => {
        const diagnostics = [];
        const output = stdout + stderr;
        const lines = output.split('\n');

        const regex = /^(.*):(\d+): (error|warning): (.*)$/;

        for (let line of lines) {
            const match = regex.exec(line.trim());
            if (match) {
                const lineNum = parseInt(match[2]) - 1;
                const severity = match[3] === 'error' ? vscode.DiagnosticSeverity.Error : vscode.DiagnosticSeverity.Warning;
                const message = match[4];

                let range = new vscode.Range(lineNum, 0, lineNum, 100);
                
                // Extraction du nom de la variable pour cibler le grisement
                const varMatch = /variable '([^']+)'/.exec(message);
                if (varMatch) {
                    const varName = varMatch[1];
                    const lineText = document.lineAt(lineNum).text;
                    const col = lineText.indexOf(varName);
                    if (col !== -1) {
                        range = new vscode.Range(lineNum, col, lineNum, col + varName.length);
                    }
                }

                const diagnostic = new vscode.Diagnostic(range, message, severity);
                
                // Appliquer le tag "Unnecessary" pour griser la variable
                if (message.includes('never used') || message.includes('inutilisé')) {
                    diagnostic.tags = [vscode.DiagnosticTag.Unnecessary];
                }
                
                diagnostics.push(diagnostic);
            }
        }
        diagnosticCollection.set(document.uri, diagnostics);
    });
}

function deactivate() {}

module.exports = {
    activate,
    deactivate
}
