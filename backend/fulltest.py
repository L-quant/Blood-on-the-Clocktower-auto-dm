#!/usr/bin/env python3
"""Run a full game test and display AI DM messages. Usage: python3 fulltest.py [provider]"""
import json, sys, time, threading, urllib.request
import websocket  # pip install websocket-client if needed

BASE = 'http://localhost:8081'
PROVIDER = sys.argv[1] if len(sys.argv) > 1 else 'gemini'

def api(method, path, data=None, token=None):
    url = f'{BASE}{path}'
    body = json.dumps(data).encode() if data else None
    req = urllib.request.Request(url, data=body, method=method)
    req.add_header('Content-Type', 'application/json')
    if token:
        req.add_header('Authorization', f'Bearer {token}')
    resp = urllib.request.urlopen(req, timeout=30)
    return json.loads(resp.read())

print(f'=== Full AutoDM Test ({PROVIDER}) ===')

# 1. Login
r = api('POST', '/v1/auth/quick', {'name': f'tester_{PROVIDER}'})
token = r['token']
user_id = r.get('user_id', '?')
print(f'[1] Logged in as {user_id}')

# 2. Create room
r = api('POST', '/v1/rooms', {'name': f'test_{PROVIDER}', 'edition': 'tb'}, token)
room_id = r['room_id']
print(f'[2] Room: {room_id}')

# 3. Join
api('POST', f'/v1/rooms/{room_id}/join', None, token)
print(f'[3] Joined room')

# 4. Add bots
api('POST', f'/v1/rooms/{room_id}/bots', {'count': 6}, token)
print(f'[4] Added 6 bots')

# 5. Connect WebSocket and start game
collected_events = []
ws_done = threading.Event()
game_started = threading.Event()

def on_message(ws, message):
    try:
        msg = json.loads(message)
        if msg.get('type') == 'event':
            evt = msg.get('payload', {})
            collected_events.append(evt)
            etype = evt.get('event_type', '?')
            if etype == 'game.started':
                game_started.set()
    except:
        pass

def on_open(ws):
    ws.send(json.dumps({
        "type": "subscribe",
        "payload": {"room_id": room_id, "last_seq": 0}
    }))
    time.sleep(0.5)
    ws.send(json.dumps({
        "type": "command",
        "payload": {"room_id": room_id, "type": "start_game"}
    }))

ws_url = f'ws://localhost:8081/ws?token={token}'
ws = websocket.WebSocketApp(ws_url, on_message=on_message, on_open=on_open)
ws_thread = threading.Thread(target=ws.run_forever, daemon=True)
ws_thread.start()

print(f'[5] WebSocket connected, waiting for game events (60s)...')
start = time.time()
timeout = 60
while time.time() - start < timeout:
    time.sleep(2)
    elapsed = int(time.time() - start)
    chat_count = sum(1 for e in collected_events if e.get('event_type') == 'public.chat')
    sys.stdout.write(f'\r    {elapsed}s elapsed, {len(collected_events)} events, {chat_count} chat messages...')
    sys.stdout.flush()
    if chat_count >= 4:
        time.sleep(5)  # collect a few more
        break

ws.close()
print(f'\n[6] Collection done: {len(collected_events)} events total')

# 6. Also fetch events via API
try:
    api_events = api('GET', f'/v1/rooms/{room_id}/events', token=token)
    print(f'[7] API returned {len(api_events)} events')
except Exception as ex:
    print(f'[7] API fetch failed: {ex}')
    api_events = []

# 7. Display results
print(f'\n{"="*60}')
print(f'  Results for {PROVIDER.upper()}')
print(f'{"="*60}')

# Combine: prefer API events (more complete), fallback to WS
events = api_events if api_events else collected_events

chat_messages = []
for e in events:
    etype = e.get('event_type', '?')
    actor = e.get('actor_user_id', '?')[:20]
    seq = e.get('seq', '?')
    pjson = e.get('payload_json', '{}')
    try:
        payload = json.loads(pjson) if isinstance(pjson, str) else pjson
    except:
        payload = {}

    if etype == 'public.chat':
        msg = payload.get('message', str(payload))
        chat_messages.append({'seq': seq, 'actor': actor, 'message': msg})
        print(f'\n[Event {seq}] {etype} by {actor}:')
        print(f'  {msg[:800]}')
    else:
        print(f'[Event {seq}] {etype} by {actor}')

# Also check WS events for chat content
if not chat_messages:
    print('\n--- Checking WebSocket collected events ---')
    for e in collected_events:
        etype = e.get('event_type', '?')
        if etype == 'public.chat':
            payload = e.get('payload', e)
            msg = payload.get('message', str(payload))
            print(f'\n[WS Chat] {msg[:800]}')

# Summary
print(f'\n{"="*60}')
print(f'  Summary ({PROVIDER.upper()})')
print(f'{"="*60}')
print(f'  Total events:  {len(events)}')
print(f'  Chat messages: {len(chat_messages)}')
print(f'  Elapsed:       {int(time.time()-start)}s')
print(f'  Room ID:       {room_id}')

# Save for comparison
result = {
    'provider': PROVIDER,
    'room_id': room_id,
    'total_events': len(events),
    'chat_count': len(chat_messages),
    'elapsed_seconds': int(time.time()-start),
    'messages': chat_messages
}
fname = f'test_result_{PROVIDER}.json'
with open(fname, 'w') as f:
    json.dump(result, f, ensure_ascii=False, indent=2)
print(f'  Results saved: {fname}')
