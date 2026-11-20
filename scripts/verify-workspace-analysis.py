"""Workspace analysis acceptance through the real CLI; run by Orrery."""
import base64
import json
import os
import subprocess
import sys
import tempfile
from html.parser import HTMLParser
from pathlib import Path

cli = Path(sys.argv[1]).resolve()
out = Path(sys.argv[2]).resolve()
out.mkdir(parents=True, exist_ok=True)
lab = Path(__file__).resolve().parents[2]


def command(*args, expected=0):
    process = subprocess.run([str(a) for a in args], text=True, capture_output=True)
    assert process.returncode == expected, (args, process.returncode, process.stdout[-3000:], process.stderr[-5000:])
    return process


command(cli, 'engine', 'install', 'go', '--from', lab / 'scryer-go-engine/build/install/scryer-go-engine', '--json')
with tempfile.TemporaryDirectory(prefix='scryer-workspace-analysis-') as directory:
    root = Path(directory).resolve()

    def write(path, source):
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(source)

    def git(repo, *args):
        return command('git', '-C', repo, *args).stdout.strip()

    def repository(name, dirty=False, source=True):
        repo = root / name
        repo.mkdir()
        git(repo, 'init', '-q')
        git(repo, 'config', 'user.name', 'Workspace fixture')
        git(repo, 'config', 'user.email', 'fixture@example.invalid')
        if source:
            write(repo / 'go.mod', 'module example.test/' + name + '\n\ngo 1.24\n')
            write(repo / 'value.go', 'package fixture\nfunc Value() int { return 1 }\nfunc Caller() int { return Value() }\nfunc Uncovered() int { return 1 }\n')
            write(repo / 'value_test.go', 'package fixture\nimport "testing"\nfunc TestValue(t *testing.T) { if Caller() < 1 { t.Fatal("wrong") } }\n')
        else:
            write(repo / 'notes.txt', 'before\n')
        git(repo, 'add', '.')
        git(repo, 'commit', '-qm', 'before')
        before = git(repo, 'rev-parse', 'HEAD')
        path = repo / ('value.go' if source else 'notes.txt')
        path.write_text(path.read_text().replace('return 1', 'return 2').replace('before', 'after'))
        if not dirty:
            git(repo, 'add', '.')
            git(repo, 'commit', '-qm', 'after')
        return repo, before, git(repo, 'rev-parse', 'HEAD')

    clean, before, after = repository('clean')
    dirty, dirty_before, _ = repository('dirty', dirty=True)

    def members(names):
        write(root / '.scryer/workspace.yaml', 'projects:\n' + ''.join('  - ' + name + '\n' for name in names))

    def analyze(name, *args, expected=0):
        process = command(cli, 'workspace', 'analyze', root, '--json', '--compact', *args, expected=expected)
        data = json.loads(process.stdout)
        (out / (name + '.json')).write_text(process.stdout)
        return data

    members(['clean', 'dirty'])
    statuses = {repo: git(repo, 'status', '--porcelain') for repo in [clean, dirty]}
    static = analyze('static', '-o', out / 'static.md')
    assert static['status'] == 'collected' and static['counts']['collected'] == 2
    reports = {p['path']: p['report'] for p in static['projects']}
    assert reports['clean']['beforeSha'] == before and reports['clean']['afterSha'] == after
    assert reports['dirty']['beforeSha'] == dirty_before and reports['dirty']['afterSource']['kind'] == 'WORKING_TREE'
    assert all(r['execution']['status'] == 'SKIPPED' and r['changes'] for r in reports.values())

    dynamic = analyze('dynamic', '--resolve', '--operations', 'impact,coverage', '-o', out / 'workspace.html')
    assert dynamic['status'] == 'collected'
    for project in dynamic['projects']:
        report = project['report']
        assert report['execution']['status'] == 'SUCCEEDED' and report['coverageReport']['status'] == 'collected'
        assert report['matching']['datasets'][0]['engine'] == 'go'
        assert any(method['status'] == 'NOT_EXECUTED' for method in report['matching']['datasets'][0]['methods'])

    class Page(HTMLParser):
        def __init__(self):
            super().__init__()
            self.frames, self.downloads = [], []
        def handle_starttag(self, tag, attributes):
            attributes = dict(attributes)
            if tag == 'iframe':
                assert attributes['sandbox'] == 'allow-scripts allow-downloads'
                self.frames.append(attributes['srcdoc'])
            if tag == 'a' and attributes.get('download') == 'workspace-analysis.json':
                self.downloads.append(json.loads(base64.b64decode(attributes['href'].split(',', 1)[1])))

    page = Page()
    page.feed((out / 'workspace.html').read_text())
    assert len(page.frames) == 2 and all('Scryer' in frame and '<script' in frame for frame in page.frames)
    assert page.downloads == [dynamic], page.downloads
    coverage = analyze('coverage', '--operations', 'coverage', '-o', out / 'coverage.html')
    assert coverage['status'] == 'collected' and all(p['reportOperation'] == 'coverage' for p in coverage['projects'])
    for repo, status in statuses.items():
        assert git(repo, 'status', '--porcelain') == status

    repository('unsupported', source=False)
    (root / 'missing-git').mkdir()
    members(['unsupported', 'missing-git', 'clean', 'dirty'])
    partial = analyze('partial', '-o', out / 'partial.md', expected=1)
    assert partial['status'] == 'partial' and partial['counts'] == {'collected': 2, 'partial': 0, 'failed': 1, 'unsupported': 1}, partial
    assert [p['status'] for p in partial['projects']] == ['unsupported', 'failed', 'collected', 'collected']
    assert all(p.get('error') for p in partial['projects'][:2])
    assert (out / 'partial.md').exists()
    members(['unsupported'])
    assert analyze('unsupported', expected=1)['status'] == 'failed'

    members(['clean', 'dirty'])
    write(dirty / 'value_test.go', 'package fixture\nimport "testing"\nfunc TestValue(t *testing.T) { t.Fatal("intentional failure") }\n')
    status = git(dirty, 'status', '--porcelain')
    failed = analyze('failed-dynamic', '--resolve', '-o', out / 'failed.html', expected=1)
    assert failed['status'] == 'partial' and failed['projects'][0]['status'] == 'collected'
    assert failed['projects'][1]['status'] == 'partial' and failed['projects'][1]['report']['after']['nodes'] and failed['projects'][1]['report']['execution']['status']=='FAILED'
    assert git(dirty, 'status', '--porcelain') == status

    existing = out / 'preserved.html'
    existing.write_text('keep existing output')
    for options in [('--before', 'HEAD~1', '--after', 'HEAD'), ('--operations', 'coverage', '--resolve'), ('--operations', 'impact,coverage')]:
        rejected = command(cli, 'workspace', 'analyze', root, *options, '-o', existing, expected=2)
        assert not rejected.stdout and existing.read_text() == 'keep existing output'
    print('Workspace static/dynamic/coverage, independent baselines, partial failures, unsupported engines, original source integrity and complete HTML/JSON: passed')
