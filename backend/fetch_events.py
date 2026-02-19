#!/usr/bin/env python3
import json, urllib.request

BASE='http://localhost:8081'

def api(method, path, data=None, token=None):
    url=f'{BASE}{path}'
    body=json.dumps(data).encode() if data else None
    req=urllib.request.Request(url,data=body,method=method)
    req.add_header('Content-Type','application/json')
    if token: req.add_header('Authorization',f'Bearer {token}')
    resp=urllib.request.urlopen(req,timeout=30)
    return json.loads(resp.read())

# Get token
r = api('POST', '/v1/auth/quick', {'name': 'viewer'})
token = r['token']

# Fetch events
import sys
room_id = sys.argv[1] if len(sys.argv) > 1 else 'cdd9aac2-6163-4c3d-974e-840166b95f08'
events = api('GET', f'/v1/rooms/{room_id}/events', token=token)
print(f'Total events: {len(events)}')
for e in events:
    etype = e.get('event_type', '?')
    actor = e.get('actor_user_id', '?')
    seq = e.get('seq', '?')
    pjson = e.get('payload_json', '{}')
    try:
        payload = json.loads(pjson) if isinstance(pjson, str) else pjson
    except:
        payload = pjson
    if etype == 'public.chat':
        msg = payload.get('message', '')
        print(f'\n[{seq}] {etype} by {actor}:')
        print(f'  {msg[:500]}')
    else:
        print(f'[{seq}] {etype} by {actor}')
