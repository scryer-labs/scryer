"""Verify engine/stage isolation and retained dynamic facts through Core."""
import json, os, subprocess, sys, tempfile
from pathlib import Path
cli,out=Path(sys.argv[1]).resolve(),Path(sys.argv[2]).resolve()
lab=Path(__file__).resolve().parents[2]
out.mkdir(parents=True,exist_ok=True)
def run(*args,code=0):
 p=subprocess.run([str(x) for x in args],text=True,capture_output=True)
 assert p.returncode==code,(args,p.returncode,p.stdout[-3000:],p.stderr[-4000:])
 return p.stdout
for engine in ['go','ts-js']:
 run(cli,'engine','install',engine,*([] if os.environ.get('SCRYER_TEST_REMOTE')=='1' else ['--from',lab/f'scryer-{engine}-engine/build/install/scryer-{engine}-engine']),'--json')
with tempfile.TemporaryDirectory(prefix='scryer-partial-analysis-') as folder:
 workspace=Path(folder).resolve();root=workspace/'mixed';root.mkdir()
 def write(name,source):
  p=root/name;p.parent.mkdir(parents=True,exist_ok=True);p.write_text(source)
 def git(*args):return run('git','-C',root,*args).strip()
 def commit(message):git('add','.');git('commit','-qm',message)
 write('go.mod','module example.test/partial\n\ngo 1.24\n')
 write('value.go','package fixture\nfunc Value() int { return 1 }\nfunc Caller() int { return Value() }\n')
 write('value_test.go','package fixture\nimport "testing"\nfunc TestValue(t *testing.T){if Caller()<1 {t.Fatal("wrong")}}\n')
 write('value.js','export function value(){return 1;}\nexport function caller(){return value();}\n')
 git('init','-q');git('config','user.name','Partial fixture');git('config','user.email','fixture@example.invalid');commit('before')
 for name in ['value.go','value.js']:
  p=root/name;p.write_text(p.read_text().replace('return 1','return 2'))
 commit('after')
 run(cli,'project','analyze',root,'-o',out/'static.html')
 def analyze(name,*options):
  raw=run(cli,'project','analyze',root,'--resolve','--operations','impact,coverage','--json','--compact',*options,code=1)
  (out/(name+'.json')).write_text(raw);d=json.loads(raw);assert d['status']=='partial',d;return d
 missing=analyze('missing-tests','-o',out/'missing-tests.html')
 by={x['engine']:x for x in missing['engineOutcomes']}
 assert by['go']['status']=='collected' and by['ts-js']['status']=='partial',by
 assert any(s['name']=='coverage' and s['status']=='unavailable' for s in by['ts-js']['stages'])
 assert {n['engine'] for n in missing['after']['nodes']}=={'go','ts-js'}
 coverage=json.loads(run(cli,'project','analyze',root,'--operations','coverage','--json','--compact'))
 assert coverage['status']=='partial' and {r['engine']:r['status'] for r in coverage['engineReports']}=={'go':'collected','ts-js':'unavailable'}
 (out/'standalone-coverage.json').write_text(json.dumps(coverage))
 (workspace/'.scryer').mkdir();(workspace/'.scryer/workspace.yaml').write_text('projects:\n  - mixed\n')
 raw=run(cli,'workspace','analyze',workspace,'--resolve','--json','--compact','-o',out/'workspace.html',code=1)
 w=json.loads(raw);assert w['status']=='partial' and w['counts']['partial']==1 and w['projects'][0]['report']['after']['nodes'],w
 (out/'workspace.json').write_text(raw)
 # Both historical inputs have a real, non-blocking type diagnostic.
 write('package.json','{"name":"partial-js","type":"module","scripts":{"test":"node --test value.test.js"}}')
 write('tsconfig.json','{"compilerOptions":{"allowJs":true,"checkJs":true,"target":"ES2022","module":"NodeNext","moduleResolution":"NodeNext","noEmit":true,"types":[]},"include":["value.js"]}')
 write('value.test.js','import {test} from "node:test"; import assert from "node:assert/strict"; import {caller} from "./value.js"; test("value",()=>assert.ok(caller()>0));\n')
 bad='/** @param {number} n */\nfunction typed(n){return n;}\ntyped("wrong");\n'
 write('value.js','export function value(){return 1;}\nexport function caller(){return value();}\n'+bad)
 commit('typed before')
 write('value.js',(root/'value.js').read_text().replace('return 1','return 2'))
 write('value.go',(root/'value.go').read_text().replace('return 2','return 3'))
 commit('typed after')
 typed=analyze('diagnostics','-o',out/'diagnostics.html')
 js=typed['extensions']['engineReports']['ts-js']
 assert js['execution']['status']=='SUCCEEDED' and js['coverageReport']['status']=='collected',js
 assert js['extensions']['beforeResolution']['status']=='partial' and js['extensions']['beforeResolution']['usableForImpact']
 assert any(e['kind']=='SCAN_RESOLVED' for e in js['after']['edges'])
 assert any(n['impact']=='CHANGED' for n in js['after']['nodes'])
 assert {x['engine']:x['status'] for x in typed['engineOutcomes']}=={'go':'collected','ts-js':'partial'}
 # A hard syntax failure loses only this engine, with no static retry.
 write('value.js',(root/'value.js').read_text()+'\nfunction broken( {')
 write('value.go',(root/'value.go').read_text().replace('return 3','return 4'))
 commit('syntax failure')
 failed=analyze('failed-engine','-o',out/'failed-engine.html')
 assert {x['engine']:x['status'] for x in failed['engineOutcomes']}=={'go':'collected','ts-js':'failed'}
 assert set(failed['extensions']['engineReports'])=={'go'} and failed['after']['nodes']
 all_failed=json.loads(run(cli,'project','analyze',root,'--stack','ts-js','--resolve','--json','--compact','-o',out/'all-failed.html',code=1))
 assert all_failed['status']=='failed' and not all_failed['after']['nodes'] and all_failed['execution']['status']=='UNAVAILABLE'
 assert git('status','--porcelain')==''
 print('Engine isolation, unavailable coverage, semantic partial, hard failure, all-failed status, workspace partial and source integrity passed.')
