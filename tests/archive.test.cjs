const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const fflate = require('../extension/vendor/fflate.js');

const context = vm.createContext({ fflate, globalThis: null });
context.globalThis = context;
vm.runInContext(fs.readFileSync(require.resolve('../extension/archive.js'), 'utf8'), context);
const { unpack } = context.WtSkinArchive;
const bytes = text => new TextEncoder().encode(text);
const archive = files => fflate.zipSync(Object.fromEntries(Object.entries(files).map(([name, data]) => [name, bytes(data)])));

test('single skin in a packaging directory', () => {
  const groups = unpack(archive({ 'UserSkins/tank/camo.blk': 'skin', 'UserSkins/tank/camo.dds': 'texture' }));
  assert.equal(groups.length, 1);
  assert.deepEqual(Array.from(groups[0].files, file => file.path.join('/')).sort(), ['camo.blk', 'camo.dds']);
});

test('multiple vehicles in separate directories', () => {
  const groups = unpack(archive({
    'pack/tank/tank.blk': 'tank', 'pack/tank/tank.dds': 'a',
    'pack/plane/plane.blk': 'plane', 'pack/plane/plane.dds': 'b'
  }));
  assert.equal(groups.length, 2);
  for (const group of groups) {
    assert.equal(group.files.filter(file => file.path.at(-1).endsWith('.blk')).length, 1);
    assert.equal(group.files.length, 2);
  }
});

test('multiple flat configs become separate skins', () => {
  const groups = unpack(archive({ 'tank.blk': 'tank', 'plane.blk': 'plane', 'shared.dds': 'texture' }));
  assert.equal(groups.length, 2);
  for (const group of groups) assert.equal(group.files.length, 2);
});

test('rejects unsafe ZIP paths', () => {
  assert.throws(() => unpack(archive({ '../escape.blk': 'bad' })), /небезопасный путь/);
});

test('rejects ZIP without camouflage config', () => {
  assert.throws(() => unpack(archive({ 'readme.txt': 'hello' })), /нет файла .blk/);
});
