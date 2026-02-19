#!/usr/bin/env python3
"""
Integrated test: Create room, add bots, start game via WebSocket, observe AI DM behavior.
Tests Gemini 3 Flash Preview AutoDM integration.
"""
import json
import time
import sys
import urllib.request
import urllib.error
import websocket
import threading

BASE_URL = "http://localhost:8081"
WS_URL = "ws://localhost:8081"

ws_messages = []

def api_call(method, path, data=None, token=None):
    url = f"{BASE_URL}{path}"
    body = json.dumps(data).encode() if data else None
    req = urllib.request.Request(url, data=body, method=method)
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", f"Bearer {token}")
    try:
        resp = urllib.request.urlopen(req, timeout=30)
        raw = resp.read().decode()
        try:
            return resp.status, json.loads(raw)
        except json.JSONDecodeError:
            return resp.status, raw
    except urllib.error.HTTPError as e:
        raw = e.read().decode()
        try:
            return e.code, json.loads(raw)
        except:
            return e.code, raw

def ws_listener(ws):
    while True:
        try:
            msg = ws.recv()
            if msg:
                try:
                    data = json.loads(msg)
                    ws_messages.append(data)
                except:
                    ws_messages.append({"raw": msg})
        except Exception:
            break

def run_test():
    global ws_messages
    ws_messages = []
    
    print("=" * 70)
    print("Blood on the Clocktower - Integrated AutoDM Test (Gemini 3 Flash)")
    print("=" * 70)
    
    # 1. Health check
    print("\n[1] Health & LLM check...")
    code, resp = api_call("GET", "/health")
    assert code == 200, f"Health failed: {code}"
    code, resp = api_call("GET", "/v1/llm/health")
    assert code == 200, f"LLM health failed: {code}"
    provider = resp.get("provider", "?")
    model = resp.get("model", "?")
    print(f"  Provider: {provider}, Model: {model}")
    
    # 2. Quick login
    print("\n[2] Authentication...")
    code, resp = api_call("POST", "/v1/auth/quick", {"name": "Gemini_Tester"})
    assert code == 200, f"Login failed: {code} {resp}"
    token = resp["token"]
    user_id = resp["user_id"]
    print(f"  User: {user_id}")
    
    # 3. Create room
    print("\n[3] Create room...")
    code, resp = api_call("POST", "/v1/rooms", {"name": "Gemini AutoDM Test", "edition": "tb"}, token)
    assert code in (200, 201), f"Create room failed: {code} {resp}"
    room_id = resp["room_id"]
    print(f"  Room: {room_id}")
    
    # 4. Join room
    print("\n[4] Join room...")
    code, resp = api_call("POST", f"/v1/rooms/{room_id}/join", None, token)
    print(f"  Join: {code}")
    
    # 5. Add bots (need 6 more for 7-player game)
    print("\n[5] Adding 6 bots...")
    for i in range(6):
        code, resp = api_call("POST", f"/v1/rooms/{room_id}/bots", {"count": 1}, token)
    print(f"  6 bots added")
    
    # 6. Connect WebSocket
    print("\n[6] Connecting WebSocket...")
    ws_url = f"{WS_URL}/ws?token={token}&room_id={room_id}"
    ws = websocket.WebSocket()
    ws.settimeout(5)
    ws.connect(ws_url)
    
    # Start listener thread
    listener = threading.Thread(target=ws_listener, args=(ws,), daemon=True)
    listener.start()
    time.sleep(1)
    
    # 7. Start game
    print("\n[7] Starting game via WebSocket...")
    start_cmd = json.dumps({"type": "start_game"})
    ws.send(start_cmd)
    
    # 8. Wait and collect AI DM messages
    print("\n[8] Waiting 30 seconds for AI DM actions...")
    for i in range(30):
        time.sleep(1)
        sys.stdout.write(f"\r  Elapsed: {i+1}s, Messages: {len(ws_messages)}  ")
        sys.stdout.flush()
    print()
    
    # 9. Analyze messages
    print(f"\n[9] Analysis: {len(ws_messages)} WebSocket messages received")
    event_types = {}
    ai_messages = []
    for msg in ws_messages:
        etype = msg.get("type", msg.get("event_type", "unknown"))
        event_types[etype] = event_types.get(etype, 0) + 1
        if "autodm" in str(msg.get("actor_user_id", "")).lower() or \
           "system" == str(msg.get("actor_user_id", "")).lower() or \
           msg.get("type") == "dm.message":
            ai_messages.append(msg)
    
    print("\n  Event type distribution:")
    for etype, count in sorted(event_types.items()):
        print(f"    {etype}: {count}")
    
    if ai_messages:
        print(f"\n  AI DM messages ({len(ai_messages)}):")
        for msg in ai_messages[:5]:
            print(f"    {json.dumps(msg, ensure_ascii=False)[:200]}")
    
    # 10. Get final game state via API
    print("\n[10] Final game state...")
    code, resp = api_call("GET", f"/v1/rooms/{room_id}/state", None, token)
    if isinstance(resp, dict):
        phase = resp.get("phase", "?")
        night = resp.get("night_count", 0)
        print(f"  Phase: {phase}, Night: {night}")
        players = resp.get("players", {})
        if isinstance(players, dict):
            alive_count = sum(1 for p in players.values() if p.get("alive"))
            dead_count = sum(1 for p in players.values() if not p.get("alive"))
            print(f"  Alive: {alive_count}, Dead: {dead_count}")
            for uid, p in players.items():
                role = p.get("role", "?")
                alive = "ALIVE" if p.get("alive") else "DEAD"
                print(f"    {uid}: {role} [{alive}]")
    
    # 11. Get events via API
    print("\n[11] Events from API...")
    code, resp = api_call("GET", f"/v1/rooms/{room_id}/events", None, token)
    events = resp if isinstance(resp, list) else resp.get("events", []) if isinstance(resp, dict) else []
    print(f"  Total events: {len(events)}")
    
    # Show last 15 events
    for e in events[-15:]:
        etype = e.get("event_type", e.get("type", "?"))
        actor = e.get("actor_user_id", "?")
        seq = e.get("seq", "?")
        payload_raw = e.get("payload", "")
        if isinstance(payload_raw, str):
            try:
                payload = json.loads(payload_raw)
            except:
                payload = payload_raw
        else:
            payload = payload_raw
        summary = str(payload)[:100]
        print(f"    [{seq}] {etype} by {actor}: {summary}")
    
    ws.close()
    
    print("\n" + "=" * 70)
    print(f"Test complete. Model: {provider}/{model}")
    print(f"  WS messages: {len(ws_messages)}, API events: {len(events)}")
    print("=" * 70)
    
    return {
        "provider": provider,
        "model": model,
        "ws_messages": len(ws_messages),
        "api_events": len(events),
        "event_types": event_types,
        "room_id": room_id,
    }

if __name__ == "__main__":
    result = run_test()
    with open("integration_test_result.json", "w") as f:
        json.dump(result, f, indent=2, ensure_ascii=False)
    print(f"\nResults saved to integration_test_result.json")
