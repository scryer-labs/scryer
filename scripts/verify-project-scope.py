"""Core + C++ project file selection acceptance, invoked by an Orrery plan.

The caller supplies an isolated SCRYER_HOME and a built C++ engine package.
No engine is installed into the user's global home.
"""
import json
import os
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

cli = Path(sys.argv[1]).resolve()
package = Path(sys.argv[2]).resolve()
out = Path(sys.argv[3]).resolve()
out.mkdir(parents=True, exist_ok=True)


def run(*args, expected=0):
    result = subprocess.run([str(a) for a in args], text=True, capture_output=True)
    assert result.returncode == expected, (args, result.returncode, result.stderr[-4000:], result.stdout[-2000:])
    return result


def write(root, name, text):
    file = root / name
    file.parent.mkdir(parents=True, exist_ok=True)
    file.write_text(text)


def scan(root, name, *args):
    response = run(cli, 'project', 'scan', root, '--facts', 'stack,loc,dependencies,architecture,files,call-graph', '--json', '--compact', *args)
    data = json.loads(response.stdout)
    (out / (name + '.json')).write_text(response.stdout)
    return data


def facts(data):
    return {f['fact']: f['data'] for f in data['engineFacts'] if f.get('engine') == 'cpp' and f['status'] == 'collected'}


def names(data):
    return {s['name'] for s in facts(data)['call-graph']['symbols']}


run(cli, 'engine', 'install', 'cpp', '--from', package, '--json')
with tempfile.TemporaryDirectory(prefix='scryer-scope-') as temporary:
    workspace = Path(temporary).resolve()
    root = workspace / 'app'
    root.mkdir()

    def git(*args):
        return run('git', '-C', root, *args).stdout.strip()

    git('init', '-q')
    git('config', 'user.name', 'Scope fixture')
    git('config', 'user.email', 'fixture@example.invalid')
    write(root, '.gitignore', 'build/\nvendor/\n')
    write(root, 'CMakeLists.txt', 'cmake_minimum_required(VERSION 3.20)\nproject(scope LANGUAGES CXX)\nadd_library(app src/app.cpp src/omit.cpp)\nenable_testing()\nadd_executable(runner tests/main.cpp)\ntarget_link_libraries(runner PRIVATE app)\nadd_test(NAME runner COMMAND runner)\n')
    write(root, 'src/app.cpp', '#include "detail.h"\nint visible(){return helper()+1;}\n')
    write(root, 'src/detail.h', 'inline int helper(){return 1;}\n')
    write(root, 'src/omit.cpp', 'int omitted(){return 4;}\n')
    write(root, 'extras/hidden.cpp', 'int hidden(){return 3;}\n')
    write(root, 'extras/hidden.cmake', 'add_library(hidden extras/hidden.cpp)\n')
    write(root, 'tests/main.cpp', 'int visible();\nint main(){return visible()>0 ? 0 : 1;}\n')
    git('add', '.')
    git('commit', '-qm', 'before')
    before = git('rev-parse', 'HEAD')
    write(root, 'src/app.cpp', '#include "detail.h"\nint visible(){return helper()+2;}\n')
    write(root, 'extras/hidden.cpp', 'int hidden(){return 5;}\n')
    git('add', '.')
    git('commit', '-qm', 'after')
    after = git('rev-parse', 'HEAD')
    write(root, 'vendor/vendor.cpp', 'int vendor(){return 6;}\n')
    baseline = scan(root, 'default')
    assert 'scope' not in baseline and {'visible', 'hidden', 'vendor'} <= names(baseline), names(baseline)
    config = 'schemaVersion: 1\nscan:\n  include: ["CMakeLists.txt", "src/**"]\n  exclude: ["src/omit.cpp"]\n'
    write(root, '.scryer/project.yaml', config)
    write(root, 'src/untracked.cpp', 'int untracked(){return 9;}\n')
    scoped = scan(root, 'scoped', '-o', out / 'scoped.md')
    scope = scoped['scope']
    assert scope['included'] == 4 and scope['excluded'] + scope['included'] == scope['considered'], scope
    assert scope['base'] == 'git-tracked-and-nonignored' and len(scope['selectionSha256']) == 64
    assert scope['include'] == ['CMakeLists.txt', 'src/**'] and scope['exclude'] == ['src/omit.cpp']
    assert names(scoped) == {'visible', 'helper', 'untracked'}, names(scoped)
    data = facts(scoped)
    assert {f['path'] for f in data['files']['files']} == {'src/app.cpp', 'src/detail.h', 'src/untracked.cpp'}
    assert {c['name'] for c in data['architecture']['components']} == {'app', 'runner'}
    assert scoped['loc']['total']['files'] == 4, scoped['loc']
    assert 'Scan scope' in (out / 'scoped.md').read_text()
    baseline_ids = {s['name']: s['symbolId'] for s in facts(baseline)['call-graph']['symbols']}
    assert all(s['symbolId'] == baseline_ids[s['name']] for s in data['call-graph']['symbols'] if s['name'] != 'untracked')
    # The compiler may read excluded files as context, but observations originate
    # only in selected source files. This does not alter the build itself.
    resolved = scan(root, 'resolved', '--resolve')
    graph = facts(resolved)['call-graph']
    assert graph['semantic']['resolution']['status'] in ('resolved', 'partial'), graph['semantic']
    assert all(e['site']['file'] != 'src/omit.cpp' for e in graph['semantic']['edges'])
    status = git('status', '--porcelain')
    impact_response = run(cli, 'project', 'analyze', root, '--before', before, '--after', after, '--json', '--compact')
    impact = json.loads(impact_response.stdout)
    (out / 'impact.json').write_text(impact_response.stdout)
    assert impact['scope']['policy'] == 'current-project-config'
    assert impact['scope']['before']['configSha256'] == scope['configSha256'] == impact['scope']['after']['configSha256']
    assert len(impact['changes']) == 1 and impact['changes'][0]['after']['signature'] == 'visible()', impact['changes']
    assert git('status', '--porcelain') == status
    dynamic_response = run(cli, 'project', 'analyze', root, '--before', before, '--after', after, '--operations', 'impact,coverage', '--resolve', '--test-target', 'runner', '--json', '--compact')
    dynamic = json.loads(dynamic_response.stdout)
    (out / 'dynamic-impact.json').write_text(dynamic_response.stdout)
    assert dynamic['status'] == 'collected' and dynamic['scope']['policy'] == 'current-project-config'
    assert len(dynamic['changes']) == 1 and dynamic['changes'][0]['after']['signature'] == 'visible()'
    assert all(n['path'] != 'tests/main.cpp' for n in dynamic['after']['nodes'])
    # The runner is outside scan selection but still executes and contributes
    # complete native coverage. Impact matches that evidence only to selected symbols.
    collection = dynamic['coverageReport']
    assert collection['status'] == 'collected' and collection['coverage']['sourceMatch'] == 'verified-snapshot'
    assert any(f['path'] == 'tests/main.cpp' for f in collection['coverage']['files']), collection
    assert git('status', '--porcelain') == status
    # Workspace reuses each project's selected report, preserving scope metadata.
    write(workspace, '.scryer/workspace.yaml', 'projects:\n  - app\n')
    response = run(cli, 'workspace', 'scan', workspace, '--facts', 'files,call-graph', '--json', '--compact')
    aggregated = json.loads(response.stdout)
    (out / 'workspace.json').write_text(response.stdout)
    assert aggregated['projects'][0]['facts']['scope']['included'] == 4, aggregated.keys()
    # An explicit empty include must never fall back to a full directory walk.
    write(root, '.scryer/project.yaml', 'schemaVersion: 1\nscan:\n  include: []\n')
    empty = scan(root, 'empty')
    assert empty['scope']['included'] == 0 and empty['loc']['total']['files'] == 0
    # No implicit path-based classification is added.
    assert not any(k in scoped for k in ('classification', 'groups'))
    # Strict validation rejects typos instead of accidentally scanning everything.
    write(root, '.scryer/project.yaml', 'schemaVersion: 1\nscan:\n  excludes: ["**/*"]\n')
    invalid = run(cli, 'project', 'scan', root, '--json', expected=1)
    assert not invalid.stdout and 'excludes' in invalid.stderr
    write(root, '.scryer/project.yaml', 'schemaVersion: 1\nscan:\n  include: ["../outside/**"]\n')
    assert 'invalid' in run(cli, 'project', 'scan', root, '--json', expected=1).stderr
    # A filesystem project has the same policy without Git inventory semantics.
    plain = workspace / 'plain'
    write(plain, 'plain.cpp', 'int plain(){return 0;}\n')
    write(plain, '.scryer/project.yaml', 'schemaVersion: 1\n')
    assert scan(plain, 'plain')['scope']['base'] == 'filesystem'
    # An older package may scan normally without config; with config it must
    # report unsupported rather than ignore the requested scope.
    old = workspace / 'old-package'
    shutil.copytree(package, old)
    manifest = json.loads((old / 'engine.json').read_text())
    manifest['capabilities'].pop('scanFileSelection')
    (old / 'engine.json').write_text(json.dumps(manifest))
    run(cli, 'engine', 'uninstall', 'cpp', '--json')
    run(cli, 'engine', 'install', 'cpp', '--from', old, '--json')
    unsupported = scan(plain, 'unsupported')
    assert all(f['status'] == 'unsupported' and 'file selection' in f['message'] for f in unsupported['engineFacts'])
    (plain / '.scryer/project.yaml').unlink()
    assert names(scan(plain, 'old-default')) == {'plain'}
print('Project scope: scan, compiler evidence, impact, workspace and unsupported-package checks passed.')
