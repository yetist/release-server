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
            "name": "mate-common",
            "version": "1.13.0",
            "tag": "v1.13.0",
            "repo": "yetist/mate-common",
            "draft": False,
            "news": "Changes since the last release: https://github.com/yetist/mate-common/compare/v1.12.0...v1.13.0\n- fix tiny typo\n- remove references to obsolete MATE components\n- pre-bump version to 1.13.0\n- release 1.13.0\n- correct NEWS a bit\n- release 1.14 release\n- release 1.15.0\n- mate-autogen: Check only for autoreconf\n\nautoconf, automake, libtool, gettext are already checked by autoreconf. Closes #19.\n\nAdapted from https://git.gnome.org/browse/gnome-common/commit/macros2/gnome-autogen.sh?id=17f56a49964a3ddabf0d166326989033b7d84764\n- Bump version to 1.15.1\n- release 1.16.0\n- update NEWS for 1.16\n- release 1.17.0\n- Update mate-common NEWS to use consistent, project wide, markdown-like formatting. This will make generating release announcements easier.\n- update NEWS for 1.18\n- release 1.18\n- pre-bump version\n- create issue_template.md\n- update issue_template\n- release 1.19.0\n- release 1.20\n- release 1.21.0\n- initial travis ci\n- release 1.22.0\n- travis: fix distcheck for fedora\n- auto release when push tag to github",
            "prerelease": False,
            'created_at' : datetime.datetime.utcnow().replace(tzinfo=datetime.timezone.utc).isoformat(),
            'published_at' : datetime.datetime.utcnow().replace(tzinfo=datetime.timezone.utc).isoformat(),
            "files": [
                {
                    "name": "mate-common-1.13.0.tar.xz",
                    "size": 69208,
                    "url": "https://github.com/yetist/mate-common/releases/download/v1.13.0/mate-common-1.13.0.tar.xz"
                    },
                {
                    "name": "mate-common-1.13.0.tar.xz.sha256sum",
                    "size": 92,
                    "url": "https://github.com/yetist/mate-common/releases/download/v1.13.0/mate-common-1.13.0.tar.xz.sha256sum"
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
    url='http://localhost:9090/release'
    #url='https://post.zhcn.cc/release'
    notify(url)
    #payload=data
    #send_post_to_url(url, payload)
