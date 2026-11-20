"""Verify installed engine analysis through the real Core CLI in an Orrery run."""
import json,os,subprocess,sys,tempfile
from pathlib import Path
cli=Path(sys.argv[1]).resolve();out=Path(sys.argv[2]).resolve();engines=sys.argv[3].split(',')
out.mkdir(parents=True,exist_ok=True)
workspace=Path(__file__).resolve().parents[2]
def command(args,**kwargs):
 p=subprocess.run([str(x) for x in args],text=True,capture_output=True,**kwargs)
 assert p.returncode==0,(args,p.stdout[-3000:],p.stderr[-5000:])
 return p.stdout
for engine in engines:
 package=workspace/f'scryer-{engine}-engine/build/install/scryer-{engine}-engine'
 command([cli,'engine','install',engine,*([] if os.environ.get('SCRYER_TEST_REMOTE') == '1' else ['--from',package]),'--json'])
with tempfile.TemporaryDirectory(prefix='scryer-native-analysis-') as folder:
 root=Path(folder).resolve()
 def write(name,text):
  p=root/name;p.parent.mkdir(parents=True,exist_ok=True);p.write_text(text)
 if 'go' in engines:
  write('go.mod','module example.test/analysis\n\ngo 1.24\n')
  write('value.go','package fixture\nfunc Value(n int) int { if n>0 { return 1 }; return 0 }\nfunc Caller() int { return Value(1) }\nfunc Uncovered() int { return 1 }\n')
  write('value_test.go','package fixture\nimport "testing"\nfunc TestValue(t *testing.T) { if Caller()<=0 { t.Fatal("wrong") } }\n')
 if 'rust' in engines:
  write('Cargo.toml','[package]\nname="analysis_fixture"\nversion="0.1.0"\nedition="2021"\n')
  write('Cargo.lock','version = 4\n[[package]]\nname = "analysis_fixture"\nversion = "0.1.0"\n')
  write('src/lib.rs','pub fn value(n:i32)->i32 { if n>0 { 1 } else { 0 } }\npub fn caller()->i32 { value(1) }\npub fn uncovered()->i32 { 1 }\n#[cfg(test)] mod tests { #[test] fn test_value() { let observed=super::caller(); assert!(observed>0); } }\n')
 if 'ts-js' in engines:
  write('package.json',json.dumps({'name':'analysis-fixture','type':'module','scripts':{'test':'node --test tests/*.test.js'}}))
  write('tsconfig.json',json.dumps({'compilerOptions':{'allowJs':True,'checkJs':False,'target':'ES2022','module':'NodeNext','moduleResolution':'NodeNext','noEmit':True},'include':['js/**/*.js','tests/**/*.js']}))
  write('js/value.js','export function value(n) { if(n>0) return 1; return 0; }\nexport function caller() { return value(1); }\nexport function uncovered() { return 1; }\n')
  write('tests/value.test.js','import { test } from "node:test";\nimport assert from "node:assert/strict";\nimport { caller } from "../js/value.js";\ntest("value",()=>assert.ok(caller()>0));\n')
 if 'cpp' in engines:
  write('CMakeLists.txt','cmake_minimum_required(VERSION 3.20)\nproject(analysis_cpp LANGUAGES CXX)\nadd_executable(cpp_tests cpp/value.cpp cpp/main.cpp)\n')
  write('cpp/value.cpp','int cpp_value(int n) { if(n>0) return 1; return 0; }\nint cpp_uncovered() { return 1; }\n')
  write('cpp/main.cpp','int cpp_value(int);\nint main() { return cpp_value(1)>0 ? 0 : 1; }\n')
 def git(*args):return command(['git','-C',root,*args]).strip()
 git('init','-q');git('config','user.name','Analysis fixture');git('config','user.email','fixture@example.invalid');git('add','.');git('commit','-qm','before');before=git('rev-parse','HEAD')
 for path in [root/'value.go',root/'src/lib.rs',root/'js/value.js',root/'cpp/value.cpp']:
  if path.exists():
   source=path.read_text().replace('return 1','return 2').replace('{ 1 }','{ 2 }')
   path.write_text(source)
 git('add','.');git('commit','-qm','after');after=git('rev-parse','HEAD')
 status=git('status','--porcelain')
 def analyze(name,*args):
  raw=command([cli,'project','analyze',root,'--json','--compact',*args]);(out/(name+'.json')).write_text(raw);return json.loads(raw)
 static=analyze('static')
 assert static['execution']['status']=='SKIPPED' and static['changes'],static
 assert static['beforeSha']==before and static['afterSha']==after
 options=['--test-target','cpp_tests'] if 'cpp' in engines else ['--resolve']
 dynamic=analyze('dynamic',*options,'--operations','impact,coverage','-o',out/'impact.html')
 assert dynamic['execution']['status']=='SUCCEEDED',dynamic
 assert dynamic['matching']['datasets'],dynamic
 assert any(n['impact']=='CALLER' for n in dynamic['after']['nodes']),dynamic['after']
 assert any(m['status'] in ('EXECUTED','PARTIALLY_EXECUTED') for d in dynamic['matching']['datasets'] for m in d['methods']),dynamic['matching']
 assert any(m['status']=='NOT_EXECUTED' and 'uncovered' in m['signature'].lower() for d in dynamic['matching']['datasets'] for m in d['methods']),dynamic['matching']
 assert len(dynamic['matching']['datasets'])==len(engines),dynamic['matching']
 if len(engines)>1:
  assert {n['engine'] for n in dynamic['after']['nodes']}==set(engines),dynamic['after']
  assert all(n['id'].startswith(n['engine']+':') for n in dynamic['after']['nodes'])
  assert len(dynamic['coverageReport']['engineReports'])==len(engines)
 coverage=analyze('coverage','--operations','coverage',*(['--test-target','cpp_tests'] if 'cpp' in engines else []),'-o',out/'coverage.md')
 assert coverage['status']=='collected',coverage
 assert git('status','--porcelain')==status and git('rev-parse','HEAD')==after
 # A failing native test must not emit an impact report or a static fallback.
 if 'go' in engines:write('value_test.go','package fixture\nimport "testing"\nfunc TestValue(t *testing.T) { t.Fatal("intentional failure") }\n')
 elif 'rust' in engines:write('src/lib.rs',(root/'src/lib.rs').read_text().replace('observed>0','observed<0'))
 elif 'ts-js' in engines:write('tests/value.test.js',(root/'tests/value.test.js').read_text().replace('caller()>0','caller()<0'))
 failure=subprocess.run([str(cli),'project','analyze',str(root),'--json',*options,'-o',str(out/'must-not-exist.html')],text=True,capture_output=True)
 assert failure.returncode!=0 and (out/'must-not-exist.html').exists(),failure
 partial=json.loads(failure.stdout)
 assert partial['status']=='partial' and partial['after']['nodes'],partial
 failed_engine='go' if 'go' in engines else 'rust' if 'rust' in engines else 'ts-js'
 child=partial.get('extensions',{}).get('engineReports',{}).get(failed_engine,partial)
 assert child['execution']['status']=='FAILED',child
 assert all(m['status'] in ('UNKNOWN','NOT_APPLICABLE') for d in child['matching']['datasets'] for m in d['methods']),child['matching']
 (out/'failure.txt').write_text(failure.stderr)
 print(json.dumps({'engines':engines,'static':'passed','dynamic':'passed','coverage':'passed','failure':'passed'}))
