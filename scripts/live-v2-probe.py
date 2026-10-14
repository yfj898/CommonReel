"""Explicit opt-in evidence. Calls the running local app and configured free API.
No credentials, raw supplier responses, or personal histories are written to reports.
"""
import argparse, hashlib, json, pathlib, secrets, time, urllib.error, urllib.parse, urllib.request
root=pathlib.Path(__file__).resolve().parents[1]
parser=argparse.ArgumentParser();parser.add_argument('--supplier',action='store_true');parser.add_argument('--model',action='store_true');parser.add_argument('--base',default='http://127.0.0.1:8093');args=parser.parse_args()
evidence=root/'docs/evidence';evidence.mkdir(exist_ok=True)
def save(name,report):
    (evidence/name).write_text(json.dumps(report,ensure_ascii=False,indent=2),encoding='utf-8')
    print(name,json.dumps(report,ensure_ascii=True),flush=True)
if args.supplier:
    env={}
    for line in (root/'.env.local').read_text(encoding='utf-8-sig').splitlines():
        if '=' in line and not line.lstrip().startswith('#'):
            k,v=line.split('=',1);env[k.strip()]=v.strip().strip('"\'')
    key=env.get('WATCHMODE_API_KEY','')
    if not key:raise SystemExit('Watchmode key absent; real supplier test NOT_RUN')
    def supplier(path,params={}):
        url='https://api.watchmode.com/v1/'+path+'?'+urllib.parse.urlencode({'apiKey':key,**params})
        try:
            with urllib.request.urlopen(url,timeout=20) as res:return res.status,json.load(res)
        except urllib.error.HTTPError as exc:return exc.code,None
        except Exception:return 0,None
    search_status,search=supplier('search/',{'search_field':'imdb_id','search_value':'tt0114709'})
    matches=[r for r in (search or {}).get('title_results',[]) if r.get('imdb_id')=='tt0114709' and r.get('type')=='movie']
    success_status,offers=(0,None)
    if len(matches)==1:success_status,offers=supplier('title/'+str(matches[0]['id'])+'/sources/',{'regions':'US'})
    failure_status,_=supplier('title/0/sources/',{'regions':'US'})
    report={'class':'real Watchmode HTTPS requests; bounded, noncommercial free-plan use','utc':time.strftime('%Y-%m-%dT%H:%M:%SZ',time.gmtime()),'region':'US','identifier_seed':'tt0114709 (Wikidata verified film)','exact_movie_matches':len(matches),'search_http':search_status,'supply_http':success_status,'offer_count':len(offers or []),'us_subscription_count':sum(x.get('region')=='US' and x.get('type')=='sub' for x in offers or []),'non_image_response_sha256':hashlib.sha256(json.dumps(offers,sort_keys=True).encode()).hexdigest(),'invalid_title_failure_http':failure_status,'raw_supplier_data_published':False,'no_real_429_claim':'Rate-limit handling uses a controlled adapter; account quota was not deliberately exhausted.'}
    save('watchmode-live-v2.json',report)
if args.model:
    state=json.loads((root/'.runtime/live-probe.json').read_text(encoding='utf-8'));secret=state['secret'];sid=state['id']
    def app(path,body=None):
        req=urllib.request.Request(args.base+'/api/v1'+path,data=json.dumps(body).encode() if body is not None else None,headers={'Content-Type':'application/json','Authorization':'Bearer '+secret})
        try:
            with urllib.request.urlopen(req,timeout=100) as res:return res.status,json.load(res)
        except urllib.error.HTTPError as exc:return exc.code,json.load(exc)
    tasks=[('explicit member veto','For Alex (member m1), set tonight\'s genre vetoes to Horror. Use get_room then set_member; keep other inputs.','m1'),('recommend and explain','Use shortlist to recommend films under the current limits. Then explain one returned candidate using explain_candidate. Do not propose a choice or record any response.','')]
    results=[]
    for label,message,speaker in tasks:
        start=time.monotonic();status,result=app('/sessions/'+sid+'/agent-turns',{'operation_key':secrets.token_hex(16),'message':message,'speaker_member_id':speaker})
        entry={'task':label,'http':status,'wall_seconds':round(time.monotonic()-start,2),'output':result};results.append(entry);print('task',label,result.get('status'),entry['wall_seconds'],flush=True)
    status,state=app('/sessions/'+sid);snapshot={'room_revision':state.get('room_revision'),'decision_version':state.get('decision_version'),'policy':state.get('policy'),'vetoes':{m['id']:m['preferences']['vetoes'] for m in state.get('members',[])}}
    save('agent-real-v2.json',{'class':'real local qwen3.5:4b + official MCP SDK + Streamable HTTP 2025-11-25; constructed group','utc':time.strftime('%Y-%m-%dT%H:%M:%SZ',time.gmtime()),'limits':{'max_tools':6,'turn_seconds':90,'context_tokens':4096},'task_count':len(tasks),'results':results,'state_after':snapshot,'phone_test':'NOT_RUN','human_group_outcomes':'NOT_TESTED'})
