from pathlib import Path
import re
root=Path(__file__).resolve().parents[1]
p=root/'docs/superpowers/plans/2026-10-02-diva-cognitive'
index=(p/'index.md').read_text()
rows=[r for r in index.splitlines() if re.match(r'\| \[S\d\d\]',r)]
deps={}
for row in rows:
 cols=row.split('|')
 sid=re.search(r'S\d\d',cols[1]).group()
 assert sid not in deps
 deps[sid]=set(re.findall(r'S\d\d',cols[4]))
assert set(deps)=={f'S{i:02}' for i in range(1,10)}
for sid,pre in deps.items(): assert sid not in pre and pre<=deps.keys()
left=dict(deps);done=set();waves=[]
while left:
 wave=sorted(k for k,v in left.items() if v<=done)
 assert wave,'Dependency cycle'
 waves.append(wave);done.update(wave)
 for k in wave:del left[k]
for sid in deps:
 s=(p/f'{sid}.md').read_text()
 for field in ['**Goal:**','**Architecture:**','**Spec:**','**Files:**','**Consumes / produces:**','**Verification:**','**Acceptance/evidence:**','- [ ]']:
  assert field in s,(sid,field)
 assert 'contracts.md' in s and 'index.md' in s
for f in root.rglob('*.md'):
 content=f.read_text()
 assert 'TBD' not in content and 'TODO' not in content, f
 for target in re.findall(r'\]\(([^)]+)\)',content):
  if '://' in target or target.startswith('#'):continue
  path=target.split('#')[0]
  assert (f.parent/path).exists(),(str(f),target)
print('PASS: nine Story files; all dependency endpoints known; no cycles; required sections and relative file links valid.')
print('Waves:',waves)
print('Product/runtime tests: NOT RUN. S08 bridge dependency remains blocked. No Story acceptance evidence manufactured.')
