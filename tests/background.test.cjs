const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

function background({ native = null, directory = null, visited = false } = {}) {
  const opened = [];
  const calls = [];
  let listener;
  const chrome = {
    runtime: {
      lastError: null,
      onMessage: { addListener(fn) { listener = fn; } },
      getURL(path) { return `chrome-extension://test/${path}`; },
      sendNativeMessage(name, message, callback) {
        calls.push(message.post);
        chrome.runtime.lastError = native ? null : { message: 'host not found' };
        callback(native);
        chrome.runtime.lastError = null;
      }
    },
    tabs: { async create(options) { opened.push(options.url); } }
  };
  const WtSkinCore = {
    async store(mode, value, key) { return key === 'setupVisited' ? visited : directory; },
    async loadPost() { return { url: 'https://live.warthunder.com/dl/test/' }; },
    async install() { return 2; }
  };
  vm.runInNewContext(fs.readFileSync(require.resolve('../extension/background.js'), 'utf8'), {
    chrome, WtSkinCore, URL, importScripts() {}
  });
  return {
    opened, calls,
    invoke: (message, url) => listener(message, { tab: { id: 1 }, url }, () => {}),
    send: (message, url = 'https://live.warthunder.com/post/1173999/en/') =>
      new Promise(resolve => listener(message, { tab: { id: 1 }, url }, resolve))
  };
}

test('installs directly through native helper without opening a tab', async () => {
  const app = background({ native: { ok: true, count: 8 } });
  assert.equal((await app.send({ type: 'INSTALL_SKIN', post: '1173999' })).installed, 8);
  assert.deepEqual(app.opened, []);
});

test('opens setup page when no helper or saved folder exists', async () => {
  const app = background();
  assert.equal((await app.send({ type: 'INSTALL_SKIN', post: '1173999' })).opened, true);
  assert.deepEqual(app.opened, ['chrome-extension://test/install.html?post=1173999']);
});

test('installs directly into a saved folder while access is granted', async () => {
  const app = background({ directory: { kind: 'directory', async queryPermission() { return 'granted'; } } });
  assert.equal((await app.send({ type: 'INSTALL_SKIN', post: '1173999' })).installed, 2);
  assert.deepEqual(app.opened, []);
});

test('reports missing helper without reopening setup after first visit', async () => {
  const app = background({ visited: true });
  const result = await app.send({ type: 'INSTALL_SKIN', post: '1173999' });
  assert.match(result.error, /Помощник не подключен/);
  assert.deepEqual(app.opened, []);
});

test('ignores messages from other sites and invalid post IDs', () => {
  const app = background();
  app.invoke({ type: 'INSTALL_SKIN', post: '1173999' }, 'https://example.com/');
  app.invoke({ type: 'INSTALL_SKIN', post: '../other' }, 'https://live.warthunder.com/');
  assert.equal(app.calls.length, 0);
  assert.equal(app.opened.length, 0);
});
