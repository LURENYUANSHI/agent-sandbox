import requests
import json

BASE = "http://localhost:8080/api/v1"
PASS = 0
FAIL = 0

def check(name, condition, detail=""):
    global PASS, FAIL
    if condition:
        print(f"  {name}: PASS")
        PASS += 1
    else:
        print(f"  {name}: FAIL - {detail[:120]}")
        FAIL += 1

for round in range(1, 6):
    print(f"=== ROUND {round} ===")

    r = requests.post(f"{BASE}/sandboxes", json={"name": f"r{round}", "policy_file": "configs/default-policy.yaml"})
    sid = r.json()["id"]
    root = r.json()["root_dir"].replace("\\", "/")

    requests.post(f"{BASE}/sandboxes/{sid}/start")

    # file:write /tmp
    r = requests.post(f"{BASE}/sandboxes/{sid}/exec", json={"type": "file:write", "params": {"path": f"/tmp/r{round}.txt", "content": f"hello round {round}"}})
    check("file:write /tmp", r.json().get("success") == True, json.dumps(r.json()))

    # file:read /tmp
    r = requests.post(f"{BASE}/sandboxes/{sid}/exec", json={"type": "file:read", "params": {"path": f"/tmp/r{round}.txt"}})
    check("file:read /tmp", r.json().get("success") == True and f"hello round {round}" in r.json().get("output", ""), json.dumps(r.json()))

    # file:write sandbox root
    r = requests.post(f"{BASE}/sandboxes/{sid}/exec", json={"type": "file:write", "params": {"path": f"{root}/data.txt", "content": "sandbox data"}})
    check("file:write sandbox", r.json().get("success") == True, json.dumps(r.json()))

    # file:read sandbox root
    r = requests.post(f"{BASE}/sandboxes/{sid}/exec", json={"type": "file:read", "params": {"path": f"{root}/data.txt"}})
    check("file:read sandbox", r.json().get("success") == True and "sandbox data" in r.json().get("output", ""), json.dumps(r.json()))

    # file:delete / (should deny)
    r = requests.post(f"{BASE}/sandboxes/{sid}/exec", json={"type": "file:delete", "params": {"path": "/"}})
    check("file:delete / DENY", r.status_code == 403, json.dumps(r.json()))

    # shell:exec (should deny)
    r = requests.post(f"{BASE}/sandboxes/{sid}/exec", json={"type": "shell:exec", "params": {"command": "whoami"}})
    check("shell:exec DENY", r.status_code == 403, json.dumps(r.json()))

    # proc:exec echo
    r = requests.post(f"{BASE}/sandboxes/{sid}/exec", json={"type": "proc:exec", "params": {"command": "echo hello-sandbox"}})
    check("proc:exec echo", r.json().get("success") == True and "hello-sandbox" in r.json().get("output", ""), json.dumps(r.json()))

    # net:http (use httpbin instead of github to avoid rate limit)
    r = requests.post(f"{BASE}/sandboxes/{sid}/exec", json={"type": "net:http", "params": {"url": "https://httpbin.org/get", "method": "GET"}})
    check("net:http", r.json().get("success") == True, json.dumps(r.json()))

    # dashboard stats
    r = requests.get(f"{BASE}/dashboard/stats")
    check("dashboard", r.json().get("total_actions", 0) > 0, json.dumps(r.json()))

    # health
    r = requests.get(f"{BASE}/health")
    check("health", r.json().get("status") == "ok", json.dumps(r.json()))

    # stop + destroy
    requests.post(f"{BASE}/sandboxes/{sid}/stop")
    requests.delete(f"{BASE}/sandboxes/{sid}")
    print()

print("=" * 40)
print(f"RESULT: {PASS} PASS / {FAIL} FAIL (total {PASS+FAIL})")
print(f"PASS RATE: {PASS/(PASS+FAIL)*100:.1f}%")
print("=" * 40)
