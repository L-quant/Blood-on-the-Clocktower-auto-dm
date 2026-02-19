#!/usr/bin/env python3
"""
LLM API 对比测试脚本
测试 Google Gemini 3 Flash Preview 和 DeepSeek API
"""
import json
import time
import sys
import urllib.request
import urllib.error

# ============== 配置 ==============
GEMINI_API_KEY = "AIzaSyDBPLTIbQGSIwjcJyanid5xNl7jLjCFvLs"
DEEPSEEK_API_KEY = "sk-361c5a8aec9143bbb49101be8b78738f"

GEMINI_BASE = "https://generativelanguage.googleapis.com/v1beta"
GEMINI_MODEL = "gemini-3-flash-preview"

DEEPSEEK_BASE = "https://api.deepseek.com/v1"
DEEPSEEK_MODEL = "deepseek-chat"

# 血染钟楼说书人 system prompt
SYSTEM_PROMPT = """你是《血染钟楼》(Blood on the Clocktower) 的AI说书人。你负责管理游戏流程，包括：
1. 夜间阶段：按顺序唤醒角色执行能力
2. 白天阶段：组织讨论、提名和投票
3. 规则裁决：根据官方规则处理各种情况
4. 氛围叙述：用生动的语言描述游戏事件

请用中文回答，语言风格要有代入感和悬疑感。"""

# 测试提示词
TEST_PROMPTS = [
    {
        "name": "基础对话 - 游戏开局叙述",
        "prompt": "现在是第一个夜晚。请为7人局（暗流涌动版本）生成一段开局夜晚的叙述。场上角色有：厨师、处女、僧侣、杀手、共情者、男爵、小恶魔。"
    },
    {
        "name": "规则判定 - 复杂情况",
        "prompt": "当前情况：小恶魔选择杀死了一名被僧侣保护的玩家。同时，毒师在夜间毒了僧侣。请判定：这名被攻击的玩家是否死亡？请给出详细的规则推理过程。"
    },
    {
        "name": "投票引导 - 提名处理",
        "prompt": "白天讨论结束。座位3号玩家（Alice）提名了座位5号玩家（Eve）。Eve声称自己是共情者，昨晚得到的数字是1。请作为说书人引导这次提名流程，包括：被提名者的辩护时间提醒、投票规则说明。"
    },
    {
        "name": "工具调用测试 - JSON结构化输出",
        "prompt": """作为说书人，现在需要执行以下操作。请以JSON格式回复你要执行的游戏命令列表：
        
当前状态：夜晚阶段，需要处理以下角色能力：
1. 毒师选择毒座位2号
2. 僧侣选择保护座位4号  
3. 小恶魔选择杀座位6号

请返回一个JSON数组，每个元素包含：action(动作类型), target(目标座位号), effect(效果描述)"""
    }
]

def test_gemini(prompt_data):
    """测试 Gemini API"""
    url = f"{GEMINI_BASE}/models/{GEMINI_MODEL}:generateContent?key={GEMINI_API_KEY}"
    
    payload = {
        "contents": [
            {"role": "user", "parts": [{"text": prompt_data["prompt"]}]}
        ],
        "systemInstruction": {
            "parts": [{"text": SYSTEM_PROMPT}]
        },
        "generationConfig": {
            "temperature": 0.7,
            "maxOutputTokens": 2048
        }
    }
    
    data = json.dumps(payload).encode('utf-8')
    req = urllib.request.Request(url, data=data, headers={"Content-Type": "application/json"})
    
    start = time.time()
    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            body = json.loads(resp.read().decode('utf-8'))
        latency = time.time() - start
        
        text = ""
        tokens = {}
        if "candidates" in body and len(body["candidates"]) > 0:
            parts = body["candidates"][0].get("content", {}).get("parts", [])
            text = "".join(p.get("text", "") for p in parts)
        if "usageMetadata" in body:
            tokens = body["usageMetadata"]
        
        return {
            "success": True,
            "text": text,
            "latency": latency,
            "tokens": tokens,
            "raw_status": 200
        }
    except urllib.error.HTTPError as e:
        latency = time.time() - start
        error_body = e.read().decode('utf-8') if e.fp else str(e)
        return {
            "success": False,
            "error": f"HTTP {e.code}: {error_body[:500]}",
            "latency": latency,
            "raw_status": e.code
        }
    except Exception as e:
        latency = time.time() - start
        return {
            "success": False,
            "error": str(e),
            "latency": latency,
            "raw_status": 0
        }

def test_deepseek(prompt_data):
    """测试 DeepSeek API"""
    url = f"{DEEPSEEK_BASE}/chat/completions"
    
    payload = {
        "model": DEEPSEEK_MODEL,
        "messages": [
            {"role": "system", "content": SYSTEM_PROMPT},
            {"role": "user", "content": prompt_data["prompt"]}
        ],
        "temperature": 0.7,
        "max_tokens": 2048
    }
    
    data = json.dumps(payload).encode('utf-8')
    req = urllib.request.Request(url, data=data, headers={
        "Content-Type": "application/json",
        "Authorization": f"Bearer {DEEPSEEK_API_KEY}"
    })
    
    start = time.time()
    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            body = json.loads(resp.read().decode('utf-8'))
        latency = time.time() - start
        
        text = ""
        tokens = {}
        if "choices" in body and len(body["choices"]) > 0:
            text = body["choices"][0].get("message", {}).get("content", "")
        if "usage" in body:
            tokens = body["usage"]
        
        return {
            "success": True,
            "text": text,
            "latency": latency,
            "tokens": tokens,
            "raw_status": 200
        }
    except urllib.error.HTTPError as e:
        latency = time.time() - start
        error_body = e.read().decode('utf-8') if e.fp else str(e)
        return {
            "success": False,
            "error": f"HTTP {e.code}: {error_body[:500]}",
            "latency": latency,
            "raw_status": e.code
        }
    except Exception as e:
        latency = time.time() - start
        return {
            "success": False,
            "error": str(e),
            "latency": latency,
            "raw_status": 0
        }

def main():
    results = {"gemini": [], "deepseek": []}
    
    print("=" * 80)
    print("🔬 血染钟楼 AutoDM - LLM API 对比测试")
    print(f"   Gemini Model:   {GEMINI_MODEL}")
    print(f"   DeepSeek Model: {DEEPSEEK_MODEL}")
    print("=" * 80)
    
    for i, prompt_data in enumerate(TEST_PROMPTS, 1):
        print(f"\n{'='*80}")
        print(f"📋 测试 {i}/{len(TEST_PROMPTS)}: {prompt_data['name']}")
        print(f"{'='*80}")
        
        # Test Gemini
        print(f"\n--- Google Gemini 3 Flash Preview ---")
        g_result = test_gemini(prompt_data)
        results["gemini"].append(g_result)
        if g_result["success"]:
            print(f"✅ 成功 | 延迟: {g_result['latency']:.2f}s | Tokens: {g_result.get('tokens', {})}")
            print(f"📝 回复 ({len(g_result['text'])} 字符):")
            print(g_result["text"][:600])
            if len(g_result["text"]) > 600:
                print(f"... (截断，总共 {len(g_result['text'])} 字符)")
        else:
            print(f"❌ 失败 | {g_result['error']}")
        
        # Test DeepSeek
        print(f"\n--- DeepSeek ---")
        d_result = test_deepseek(prompt_data)
        results["deepseek"].append(d_result)
        if d_result["success"]:
            print(f"✅ 成功 | 延迟: {d_result['latency']:.2f}s | Tokens: {d_result.get('tokens', {})}")
            print(f"📝 回复 ({len(d_result['text'])} 字符):")
            print(d_result["text"][:600])
            if len(d_result["text"]) > 600:
                print(f"... (截断，总共 {len(d_result['text'])} 字符)")
        else:
            print(f"❌ 失败 | {d_result['error']}")
    
    # Summary
    print(f"\n{'='*80}")
    print("📊 综合对比总结")
    print(f"{'='*80}")
    
    for provider in ["gemini", "deepseek"]:
        provider_results = results[provider]
        successes = sum(1 for r in provider_results if r["success"])
        avg_latency = sum(r["latency"] for r in provider_results if r["success"]) / max(successes, 1)
        avg_len = sum(len(r.get("text", "")) for r in provider_results if r["success"]) / max(successes, 1)
        
        name = "Gemini 3 Flash Preview" if provider == "gemini" else "DeepSeek Chat"
        print(f"\n🤖 {name}:")
        print(f"   成功率: {successes}/{len(provider_results)}")
        print(f"   平均延迟: {avg_latency:.2f}s")
        print(f"   平均回复长度: {avg_len:.0f} 字符")
    
    # Save full results
    with open("/Users/qingchang/Blood-on-the-Clocktower-auto-dm/backend/llm_test_results.json", "w", encoding="utf-8") as f:
        # Convert results for JSON serialization
        for provider in results:
            for r in results[provider]:
                if "tokens" in r:
                    r["tokens"] = dict(r["tokens"]) if hasattr(r["tokens"], "items") else r["tokens"]
        json.dump(results, f, ensure_ascii=False, indent=2)
    
    print("\n📁 完整结果已保存到 backend/llm_test_results.json")
    
    # Return exit code
    all_gemini_ok = all(r["success"] for r in results["gemini"])
    all_deepseek_ok = all(r["success"] for r in results["deepseek"])
    if not all_gemini_ok:
        print("\n⚠️  Gemini 测试存在失败项，需要检查和修复")
    if not all_deepseek_ok:
        print("\n⚠️  DeepSeek 测试存在失败项，需要检查和修复")
    
    return 0 if (all_gemini_ok and all_deepseek_ok) else 1

if __name__ == "__main__":
    sys.exit(main())
