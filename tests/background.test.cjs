const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

function background() {
  const opened = [];
  let listener;
  const chrome = {
    runtime: {
      onMessage: { addListener(fn) { listener = fn; } },
      getURL(path) { return `chrome-extension://test/${path}`; }
    },
    tabs: { create(options) { opened.push(options.url); } }
  };
  vm.runInNewContext(fs.readFileSync(require.resolve('../extension/background.js'), 'utf8'), { chrome, URL });
  return { opened, send: (message, url) => listener(message, { tab: { id: 1 }, url }) };
}

test('opens installer from WT Live through extension API', () => {
  const app = background();
  app.send({ type: 'OPEN_INSTALLER', post: '1173999' }, 'https://live.warthunder.com/post/1173999/en/');
  assert.deepEqual(app.opened, ['chrome-extension://test/install.html?post=1173999']);
});

test('rejects messages from other sites and invalid post IDs', () => {
  const app = background();
  app.send({ type: 'OPEN_INSTALLER', post: '1173999' }, 'https://example.com/');
  app.send({ type: 'OPEN_INSTALLER', post: '../other' }, 'https://live.warthunder.com/');
  assert.equal(app.opened.length, 0);
});
