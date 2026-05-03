const vscode = require('vscode');
const cp = require('child_process');
const path = require('path');

let diagnosticCollection;

function activate(context) {
    diagnosticCollection = vscode.languages.createDiagnosticCollection('xru');
    context.subscriptions.push(diagnosticCollection);

    // DÉSACTIVÉ : L'exécution automatique de XRU est dangereuse car XRU modifie les fichiers.
    // L'extension ne doit faire que de la coloration syntaxique pour l'instant.
    /*
    context.subscriptions.push(vscode.workspace.onDidSaveTextDocument(doc => {
        if (doc.languageId === 'xru') {
            updateDiagnostics(doc);
        }
    }));
    */
}

function updateDiagnostics(document) {
    // Cette fonction est désactivée pour éviter les modifications de fichiers involontaires.
}

function deactivate() {}

module.exports = {
    activate,
    deactivate
}
