#!/usr/bin/env python3
import json

with open('test_result_gemini.json') as f:
    g = json.load(f)
with open('test_result_deepseek.json') as f:
    d = json.load(f)

print('=' * 70)
print('  GEMINI vs DEEPSEEK Comparison')
print('=' * 70)

print(f'\nMetric               Gemini              DeepSeek')
print('-' * 65)
print(f'Model                gemini-3-flash      deepseek-chat')
print(f'Total Events         {g["total_events"]:<20}{d["total_events"]}')
print(f'Elapsed (sec)        {g["elapsed_seconds"]:<20}{d["elapsed_seconds"]}')

print('\n' + '=' * 70)
print('  GEMINI AI DM Messages')
print('=' * 70)
for i, m in enumerate(g.get('messages', []), 1):
    msg = m.get('message', str(m))
    print(f'\n--- Gemini Msg {i} ---')
    print(msg[:800])

print('\n' + '=' * 70)
print('  DEEPSEEK AI DM Messages')
print('=' * 70)
for i, m in enumerate(d.get('messages', []), 1):
    msg = m.get('message', str(m))
    print(f'\n--- DeepSeek Msg {i} ---')
    print(msg[:800])
