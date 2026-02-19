#!/usr/bin/env python3
import json, urllib.request, urllib.error, websocket, time, sys

BASE='http://localhost:8081'

def api(method, path, data=None, token=None):
    url=f'{BASE}{path}'
    body=json.dumps(data).encode() if data else None
    req=urllib.request.Request(url,data=body,method=method)
    req.add_header('Content-Type','application/json')
    if token: req.add_header('Authorization',f'Bearer {token}')
    try:
        resp=urllib.request.urlopen(req,timeout=30)
        return resp.status, json.loads(resp.read())
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode()

print("=" * 70)
print("Gemini 3 Flash Preview - Full Integration Test")
print("=" * 70)

# Login
_,r=api('POST','/v1/auth/quick',{'name':'Gemini_Player'})
token=r['token']; uid=r['user_id']
print(f'User: {uid}')

# Create & join
_,r=api('POST','/v1/rooms',{'name':'Gemini Test','edition':'tb'},token)
rid=r['room_id']
print(f'Room: {rid}')
api('POST',f'/v1/rooms/{rid}/join',None,token)

# Add 6 bots
for i in range(6):
    api('POST',f'/v1/rooms/{rid}/bots',{'count':1},token)
print('7 players ready (1 human + 6 bots)')

# Connect WS
ws=websocket.WebSocket()
ws.settimeout(10)
ws.connect(f'ws://localhost:8081/ws?token={token}&room_id={rid}')

# Subscribe
ws.send(json.dumps({'type':'subscribe','request_id':'sub1','payload':{'room_id':rid,'last_seq':0}}))
# Drain initial events
events = []
while True:
    try:
        ws.settimeout(2)
        msg = json.loads(ws.recv())
        events.append(msg)
        if msg.get('type') == 'subscribed':
            break
    except:
        break
print(f'Subscribed. Initial events: {len(events)}')

# Start game
ws.settimeout(15)
ws.send(json.dumps({'type':'command','request_id':'start1','payload':{'room_id':rid,'type':'start_game'}}))

# Collect all events for 45 seconds
print('Game started. Collecting events for 45 seconds...')
all_events = []
start_time = time.time()
while time.time() - start_time < 45:
    try:
        ws.settimeout(3)
        raw = ws.recv()
        msg = json.loads(raw)
        all_events.append(msg)
        etype = msg.get('type','?')
        if etype == 'event':
            payload = msg.get('payload',{})
            event_type = payload.get('event_type','?')
            data = payload.get('data',{})
            # Print AI DM messages
            if event_type in ('dm.public_message', 'dm.private_message', 'ai.decision', 'autodm.message'):
                content = data.get('content', data.get('message', str(data)))[:120]
                print(f'  [AI] {event_type}: {content}')
            elif event_type in ('game.started', 'phase.first_night', 'phase.day', 'phase.night', 'game.ended'):
                print(f'  [PHASE] {event_type}')
            elif event_type in ('player.died', 'execution.resolved', 'demon.changed'):
                print(f'  [EVENT] {event_type}: {data}')
        elif etype == 'command_result':
            p = msg.get('payload',{})
            print(f'  [CMD] {p.get("status","?")} seq {p.get("applied_seq_from","?")}-{p.get("applied_seq_to","?")}')
    except websocket.WebSocketTimeoutException:
        continue
    except Exception as e:
        print(f'  [ERR] {e}')
        break

elapsed = time.time() - start_time
print(f'\nCollected {len(all_events)} events in {elapsed:.0f}s')

# Analyze
event_types = {}
for msg in all_events:
    if msg.get('type') == 'event':
        et = msg.get('payload',{}).get('event_type','?')
        event_types[et] = event_types.get(et, 0) + 1
print('\nEvent distribution:')
for et, cnt in sorted(event_types.items()):
    print(f'  {et}: {cnt}')

# Get final state
_,st=api('GET',f'/v1/rooms/{rid}/state',None,token)
print(f'\nFinal state: phase={st.get("phase","?")}, night={st.get("night_count",0)}')
for k,v in st.get('players',{}).items():
    role = v.get('role','?')
    alive = 'ALIVE' if v.get('alive') else 'DEAD'
    print(f'  {k}: {role} [{alive}]')

ws.close()
print('\n' + '=' * 70)
