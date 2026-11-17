const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

const context = vm.createContext({ globalThis: null });
context.globalThis = context;
vm.runInContext(fs.readFileSync(require.resolve('../extension/core.js'), 'utf8'), context);
const { installationFolderName } = context.WtSkinCore;

test('uses archive title for a single skin', () => {
  assert.equal(installationFolderName("Maus, 'Alles Kaputt', 1945.zip", 'germ_pzkpfw_Maus', 1), "Maus, 'Alles Kaputt', 1945");
});

test('names every skin in a multi-vehicle pack', () => {
  assert.equal(installationFolderName('Blackout_10.3+10.7_Germany.zip', 'Blackout_boxer_3105', 8), 'Blackout 10.3+10.7 Germany - Blackout boxer 3105');
});

test('removes invalid folder characters', () => {
  assert.equal(installationFolderName('A:B?C.zip', 'x', 1), 'A B C');
});
