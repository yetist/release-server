#!/usr/bin/python3
# -*- encoding:utf-8 -*-
# vim: set filetype=python

import os
import sys
import json
import requests
import hmac
import hashlib
import uuid
import datetime

def send_post_to_url(url, payload, headers={}):
    nonce = str(uuid.uuid4()).replace('-','').upper()
    if type(payload) == str:
        body = payload
    else:
        body = json.dumps(payload, indent=2, default=str)

    secret = os.getenv('API_SECRET', '')
    if not secret:
        print('Please set the "API_SECRET" environment variable for secure transfer.')
    else:
        data = nonce + body
        signature = hmac.new(bytes(secret, 'utf8'), msg = bytes(data, 'utf8'), digestmod = hashlib.sha256)
        sign = signature.hexdigest().upper()
        headers['X-Build-Signature'] = sign

    headers['X-Build-Nonce'] = nonce
    try:
        r = requests.post(url, data=body, headers=headers)
        if r.status_code != 200:
            print('Visit {} : {} {}'.format(url, r.status_code, r.reason))
            return 1
        else:
            print(r.text)
            print('%s has been notified' % url)
            return 0
    except requests.exceptions.ConnectionError as error:
        print('Connect Error {} : {}'.format(url, error))
        return 1

def notify(url):
    data = {
            "name": "docker-build",
            "version": "1.3.3",
            "tag": "1.3.3",
            "repo": "yetist/docker-build",
            "draft": False,
            "news": "Changes since the last release: https://github.com/yetist/docker-build/compare/v1.2.20...1.3.0\n- The commit title\n\nThe commit body\nline 2\nline 3\nline 4\n- Adfs23432f Title\n\nContent 1\nline 2\nline 3\n&asdf #sfdsfs\n$1\nthe end line",
            "prerelease": False,
            'created_at' : datetime.datetime.utcnow().replace(tzinfo=datetime.timezone.utc).isoformat(),
            'published_at' : datetime.datetime.utcnow().replace(tzinfo=datetime.timezone.utc).isoformat(),
            "files": [
                {
                    "name": "distro_hook",
                    "size": 1330,
                    "url": "https://github.com/yetist/docker-build/releases/download/v1.3.3/distro_hook"
                },
                {
                    "name": "distro_hook.sha256sum",
                    "size": 78,
                    "url": "https://github.com/yetist/docker-build/releases/download/v1.3.3/distro_hook.sha256sum"
                },
                {
                    "name": "NEWS",
                    "size": 6396,
                    "url": "https://github.com/yetist/docker-build/releases/download/v1.3.3/NEWS"
                },
                {
                    "name": "NEWS.sha256sum",
                    "size": 71,
                    "url": "https://github.com/yetist/docker-build/releases/download/v1.3.3/NEWS.sha256sum"
                }
            ]
        }
    payload = json.dumps(data, indent=2, default=str)
    result = send_post_to_url(url, payload)
    if result > 0:
        print('We can not send post to all urls')
        sys.exit(1)
    else:
        print('Notification has send')

if __name__=="__main__":
    url='http://localhost:8080/post'
    notify(url)
    #payload=data
    #send_post_to_url(url, payload)
