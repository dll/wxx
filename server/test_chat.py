#!/usr/bin/env python3
# -*- coding: utf-8 -*-
import requests
import json

# 登录获取 token
login_resp = requests.post(
    'http://localhost:8081/api/v1/auth/login',
    json={'username': 'test_student', 'password': 'password123'},
    headers={'Content-Type': 'application/json; charset=utf-8'}
)
token = login_resp.json()['data']['token']
print(f"登录成功，token: {token[:50]}...")

# 发送问答请求
chat_resp = requests.post(
    'http://localhost:8081/api/v1/chat',
    json={'question': '国家奖学金的申请条件是什么？'},
    headers={
        'Content-Type': 'application/json; charset=utf-8',
        'Authorization': f'Bearer {token}'
    }
)

result = chat_resp.json()
print(f"\n响应状态: {chat_resp.status_code}")
print(f"session_id: {result.get('session_id')}")
print(f"trace_id: {result['data']['trace_id']}")
print(f"confidence: {result['data']['confidence']}")
print(f"fallback: {result['data']['fallback']}")
print(f"sources 数量: {len(result['data']['sources']) if result['data']['sources'] else 0}")
print(f"\n回答:\n{result['data']['conclusion']}")

if result['data']['sources']:
    print("\n来源:")
    for src in result['data']['sources']:
        print(f"  - {src['title']} (relevance: {src['relevance_score']:.2f})")
