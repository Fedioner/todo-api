#!/usr/bin/env python3
"""Verify the HTTP contract against a running Todo API."""
import argparse
import json
from urllib.error import HTTPError
from urllib.request import Request, urlopen
from uuid import uuid4


def request(base_url, method, path, status, payload=None, version=None):
    data = None if payload is None else json.dumps(payload).encode("utf-8")
    headers = {} if data is None else {"Content-Type": "application/json"}
    req = Request(base_url + path, data=data, headers=headers, method=method)
    try:
        response = urlopen(req, timeout=5)
    except HTTPError as error:
        response = error
    with response:
        body = response.read()
        if response.status != status:
            raise RuntimeError(f"{method} {path}: HTTP {response.status}, expected {status}")
        if version and response.headers.get("X-App-Version") != version:
            raise RuntimeError(f"{method} {path}: unexpected application version")
    return json.loads(body) if body else None


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("url", help="URL of the running API")
    parser.add_argument("--version", help="expected value of X-App-Version")
    parser.add_argument("--read-only", action="store_true", help="check health and listing only")
    args = parser.parse_args()
    base = args.url.rstrip("/")
    for path in ("/livez", "/readyz"):
        request(base, "GET", path, 200, version=args.version)
    tasks = request(base, "GET", "/api/tasks", 200, version=args.version)
    if not isinstance(tasks, list):
        raise RuntimeError("expected a list of tasks")
    if args.read_only:
        print("Health checks and task listing passed.")
        return

    title = f"Smoke {uuid4()}"
    task = request(base, "POST", "/api/tasks", 201, {"title": title})
    path = f"/api/tasks/{task['id']}"
    try:
        if task["title"] != title or task["completed"] is not False:
            raise RuntimeError("created task does not match input")
        tasks = request(base, "GET", "/api/tasks", 200)
        if not any(item["id"] == task["id"] for item in tasks):
            raise RuntimeError("created task missing from list")
        task = request(base, "PATCH", path, 200, {"completed": True})
        if task["completed"] is not True or task["title"] != title:
            raise RuntimeError("task was not completed")
    finally:
        request(base, "DELETE", path, 204)
    request(base, "PATCH", path, 404, {"completed": False})
    print("Health checks and task lifecycle passed.")


if __name__ == "__main__":
    main()
