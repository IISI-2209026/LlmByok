#!/usr/bin/env python3
"""
Create a GitHub Pull Request via the REST API.

Why a Python script rather than PowerShell?
  PowerShell 5.1's ConvertTo-Json mangles non-ASCII (Traditional Chinese)
  and GitHub strips HTML-like tags (e.g. `<version>`) from PR bodies, which
  produced garbled PR descriptions. Python handles UTF-8 JSON natively and
  reliably, so the PR title/body render correctly.

Usage:
  python create_pr.py --owner OWNER --repo REPO --head HEAD --base BASE \
                      --title-file TITLE_FILE --body-file BODY_FILE \
                      [--update-existing PR_NUMBER]

Auth:
  The script reads a GitHub token from `git credential fill` for
  host=github.com, so it reuses whatever credential helper the local git
  is configured with (Windows Credential Manager, osxkeychain, etc.).
  It never prints the token.

Inputs:
  --title-file : UTF-8 text file whose entire content is the PR title.
  --body-file  : UTF-8 text file whose entire content is the PR body
                 (GitHub markdown). Use {placeholder} instead of
                 <placeholder> to avoid GitHub stripping HTML-like tags.

If --update-existing PR_NUMBER is given, the script PATCHes an existing PR
(title + body) instead of creating a new one. This is useful when a PR was
created with a garbled body and needs to be fixed.
"""
import argparse
import json
import subprocess
import sys
import urllib.request
import urllib.error


def get_token() -> str:
    cred_input = b'protocol=https\nhost=github.com\n\n'
    cred = subprocess.run(
        ['git', 'credential', 'fill'],
        input=cred_input,
        capture_output=True,
    )
    out = cred.stdout.decode('utf-8', errors='replace')
    for line in out.splitlines():
        if line.startswith('password='):
            token = line.split('=', 1)[1].strip()
            if token:
                return token
    print('ERROR: no GitHub token available from `git credential fill`.',
          file=sys.stderr)
    print('Run `git credential fill` manually to verify a credential is '
          'stored for https://github.com.', file=sys.stderr)
    sys.exit(1)


def read_file(path: str) -> str:
    with open(path, encoding='utf-8') as f:
        return f.read()


def main() -> None:
    parser = argparse.ArgumentParser(description='Create or update a GitHub PR')
    parser.add_argument('--owner', required=True)
    parser.add_argument('--repo', required=True)
    parser.add_argument('--head', required=True, help='Head branch (source)')
    parser.add_argument('--base', required=True, help='Base branch (target)')
    parser.add_argument('--title-file', required=True,
                        help='UTF-8 file containing the PR title')
    parser.add_argument('--body-file', required=True,
                        help='UTF-8 file containing the PR body (markdown)')
    parser.add_argument('--update-existing', type=int, default=None,
                        help='PATCH an existing PR by number instead of creating')
    args = parser.parse_args()

    title = read_file(args.title_file).strip()
    body = read_file(args.body_file).strip()

    if not title:
        print('ERROR: title is empty', file=sys.stderr)
        sys.exit(1)

    token = get_token()
    headers = {
        'Authorization': f'Bearer {token}',
        'Accept': 'application/vnd.github+json',
        'X-GitHub-Api-Version': '2022-11-28',
        'Content-Type': 'application/json; charset=utf-8',
    }

    if args.update_existing is not None:
        url = (f'https://api.github.com/repos/{args.owner}/{args.repo}'
               f'/pulls/{args.update_existing}')
        payload = {'title': title, 'body': body}
        method = 'PATCH'
        action = 'updated'
    else:
        url = (f'https://api.github.com/repos/{args.owner}/{args.repo}'
               f'/pulls')
        payload = {'title': title, 'head': args.head, 'base': args.base,
                   'body': body}
        method = 'POST'
        action = 'created'

    data = json.dumps(payload, ensure_ascii=False).encode('utf-8')
    req = urllib.request.Request(url, method=method, data=data, headers=headers)

    try:
        with urllib.request.urlopen(req) as resp:
            result = json.loads(resp.read().decode('utf-8'))
    except urllib.error.HTTPError as e:
        body_text = e.read().decode('utf-8', errors='replace')
        print(f'ERROR: HTTP {e.code} {e.reason}', file=sys.stderr)
        print(body_text, file=sys.stderr)
        sys.exit(1)

    print(f'PR #{result["number"]} {action}: {result["html_url"]}')


if __name__ == '__main__':
    main()