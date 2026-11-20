// Execute generated report scripts in an isolated DOM, without loading resources.
const fs = require('node:fs');
const path = require('node:path');
const assert = require('node:assert/strict');
const {JSDOM, VirtualConsole} = require(require.resolve('jsdom', {paths: [process.argv[2]]}));
const directory = process.argv[3];
for (const name of ['static.html', 'missing-tests.html', 'diagnostics.html', 'failed-engine.html', 'all-failed.html']) {
  const errors = [];
  const virtualConsole = new VirtualConsole();
  virtualConsole.on('jsdomError', error => errors.push(error.message));
  const dom = new JSDOM(fs.readFileSync(path.join(directory, name), 'utf8'), {
    runScripts: 'dangerously', pretendToBeVisual: true, virtualConsole,
    beforeParse(window) { window.ResizeObserver = class {observe() {} disconnect() {}}; }
  });
  assert.deepEqual(errors, [], name);
  assert.ok(dom.window.ScryerReport, `${name}: application initialized`);
  assert.ok(dom.window.document.querySelector('#analysis-outcomes').textContent.includes('Analysis:'), name);
  dom.window.document.querySelector('#tab-impact').click();
  assert.equal(dom.window.ScryerReport.state.panel, 'impact', name);
  assert.deepEqual(errors, [], name);
  dom.window.close();
}
console.log('Generated static, partial and failed report scripts initialized and switched to impact view.');
