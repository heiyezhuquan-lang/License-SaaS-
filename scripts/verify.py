#!/usr/bin/env python3
import hashlib, hmac, json, time, urllib.error, urllib.request
BASE = 'http://127.0.0.1:8080'
CLIENT_SECRET = 'demo-client-secret-change-me'

_counter = 0

def sign_headers(method, path, body_text):
    global _counter
    _counter += 1
    ts = str(int(time.time()))
    nonce = f'verify-{ts}-{_counter}'
    canonical = method + '\n' + path + '\n' + ts + '\n' + nonce + '\n' + body_text
    sig = hmac.new(CLIENT_SECRET.encode(), canonical.encode(), hashlib.sha256).hexdigest()
    return {'X-Timestamp': ts, 'X-Nonce': nonce, 'X-Signature': sig, 'X-App-Key': 'demo-app'}

def req(path, method='GET', data=None, headers=None, expect_error=False, signed=False, nonce=None):
    body_text = json.dumps(data, separators=(',', ':')) if data is not None else ''
    body = body_text.encode() if data is not None else None
    h = {'Content-Type':'application/json'}
    if signed:
        h.update(sign_headers(method, path, body_text))
        if nonce:
            ts = h['X-Timestamp']
            canonical = method + '\n' + path + '\n' + ts + '\n' + nonce + '\n' + body_text
            h['X-Nonce'] = nonce
            h['X-Signature'] = hmac.new(CLIENT_SECRET.encode(), canonical.encode(), hashlib.sha256).hexdigest()
    if headers: h.update(headers)
    r = urllib.request.Request(BASE+path, data=body, method=method, headers=h)
    try:
        with urllib.request.urlopen(r, timeout=10) as resp:
            text = resp.read().decode()
            return json.loads(text or '{}')
    except urllib.error.HTTPError as e:
        payload = e.read().decode()
        if expect_error:
            try: return json.loads(payload or '{}')
            except Exception: return {'ok': False, 'status': e.code, 'message': payload}
        raise

def client(path, method='GET', data=None, headers=None, expect_error=False, nonce=None):
    return req(path, method, data, headers, expect_error, signed=True, nonce=nonce)

print('[1] health'); req('/api/health')
print('[1b] unsigned client request rejected')
unsigned = req('/api/client/app-info?app_key=demo-app', expect_error=True)
assert unsigned.get('ok') is False, unsigned
print('[2] setup status/admin login')
setup = req('/api/setup/status')
assert setup.get('installed') is True, setup
setup_again = req('/api/setup/admin', 'POST', {'username':'verify-admin','password':'verify-pass'}, expect_error=True)
assert setup_again.get('ok') is False, setup_again
login = req('/api/admin/login', 'POST', {'username':'admin','password':'admin123'})
auth_header = {'Authorization':'Bearer '+login['token']}
print('[3] apps/stats');
apps = req('/api/admin/apps', headers=auth_header)['data']; req('/api/admin/stats', headers=auth_header)
app1 = next(x for x in apps if x['app_key'] == 'demo-app')
CLIENT_SECRET = app1['client_secret']
print('[3a] signed replay rejected')
replay_nonce = 'replay-' + str(int(time.time()))
client('/api/client/app-info?app_key=demo-app', nonce=replay_nonce)
replayed = client('/api/client/app-info?app_key=demo-app', expect_error=True, nonce=replay_nonce)
assert replayed.get('ok') is False, replayed
req(f"/api/admin/apps/{app1['id']}", 'PUT', {'name':app1['name'],'status':'active','version':'1.2.3','minVersion':'1.0.0','forceUpdate':1,'forceUpdateMessage':'请更新到最新版本','downloadURL':'https://example.com/main.exe','backupDownloadURL':'https://example.com/backup.exe','announcement':'验证公告','heartbeatInterval':45,'heartbeatTimeout':135}, auth_header)
info = client('/api/client/app-info?app_key=demo-app')['data'][0]
assert info['version'] == '1.2.3' and info['min_version'] == '1.0.0' and int(info['force_update']) == 1 and info['heartbeat_interval'] == 45 and info['heartbeat_timeout'] == 135, info
print('[3b] cloud vars admin CRUD and signed client read')
suffix = str(int(time.time()))
kv = [
    ('notice_text_'+suffix, '欢迎使用云配置'),
    ('enable_feature_x_'+suffix, 'true'),
    ('max_retry_'+suffix, '3'),
    ('json_config_'+suffix, '{"level":2,"tags":["a","b"]}'),
]
ids = []
for key, val in kv:
    req('/api/admin/cloud-vars', 'POST', {'appId': app1['id'], 'varKey': key, 'varValue': val, 'valueType': 'number', 'status': 'active', 'remark': 'verify'}, auth_header)
rows = req('/api/admin/cloud-vars?keyword='+suffix, headers=auth_header)['data']
assert len(rows) >= 4, rows
assert all(x.get('value_type') == 'text' for x in rows), rows
cloud_without_token = client('/api/client/cloud-vars?app_key=demo-app', expect_error=True)
assert cloud_without_token.get('ok') is False and cloud_without_token.get('message') == '登录令牌无效或已过期', cloud_without_token
row0 = next(x for x in rows if x['var_key'] == 'notice_text_'+suffix)
req(f"/api/admin/cloud-vars/{row0['id']}", 'PUT', {'varKey': row0['var_key'], 'varValue': '已更新', 'valueType': 'text', 'status': 'active', 'remark': 'verify update'}, auth_header)
print('[4] admin generate hour-based card')
missing_type = req('/api/admin/cards/generate','POST',{'appId':1,'count':1},auth_header,expect_error=True)
assert missing_type.get('ok') is False and '卡类' in missing_type.get('message',''), missing_type
hour_type = req('/api/admin/card-types','POST',{'appId':1,'name':'2小时体验卡','hours':2,'maxDevices':1,'price':1.5,'freeUnbinds':1,'maxUnbinds':2,'unbindDeductHours':1},auth_header)
type_rows = req('/api/admin/card-types', headers=auth_header)['data']
hour_row = next(x for x in type_rows if x['name'] == '2小时体验卡')
assert int(hour_row['hours']) == 2, hour_row
assert int(hour_row['free_unbinds']) == 1 and int(hour_row['max_unbinds']) == 2 and int(hour_row['unbind_deduct_hours']) == 1, hour_row
card = req('/api/admin/cards/generate','POST',{'appId':1,'cardTypeId':hour_type['id'],'count':1},auth_header)['data'][0]
standalone_card = req('/api/admin/cards/generate','POST',{'appId':1,'cardTypeId':hour_type['id'],'count':1},auth_header)['data'][0]
recharge_card = req('/api/admin/cards/generate','POST',{'appId':1,'cardTypeId':hour_type['id'],'count':1},auth_header)['data'][0]
card_rows = req('/api/admin/cards?keyword='+card, headers=auth_header)['data']
assert int(card_rows[0]['expire_hours']) == 2, card_rows[0]
assert int(card_rows[0]['free_unbinds']) == 1 and int(card_rows[0]['max_unbinds']) == 2 and int(card_rows[0]['unbind_deduct_hours']) == 1, card_rows[0]
print('card='+card)
print('[5] signed client register/login/recharge/heartbeat')
user = 'u' + str(int(time.time()))
register_without_card = client('/api/client/register','POST',{'appKey':'demo-app','username':user+'_nocard','password':'123456'}, expect_error=True)
assert register_without_card.get('ok') is False, 'register without card should be rejected'
client('/api/client/register','POST',{'appKey':'demo-app','username':user,'password':'123456','cardKey':card,'machineCode':'PC-001'})
client_login = client('/api/client/login','POST',{'appKey':'demo-app','username':user,'password':'123456','machineCode':'PC-001'})
assert client_login['heartbeat_interval'] == 45 and client_login['heartbeat_timeout'] == 135, client_login
cloud = client('/api/client/cloud-vars?app_key=demo-app', headers={'Authorization':'Bearer '+client_login['client_token']})
assert cloud['data']['notice_text_'+suffix] == '已更新', cloud
assert cloud['data']['enable_feature_x_'+suffix] == 'true', cloud
assert cloud['data']['max_retry_'+suffix] == '3', cloud
assert cloud['data']['json_config_'+suffix] == '{"level":2,"tags":["a","b"]}', cloud
client('/api/client/recharge','POST',{'appKey':'demo-app','username':user,'cardKey':recharge_card})
standalone_login = client('/api/client/card-login','POST',{'appKey':'demo-app','cardKey':standalone_card,'machineCode':'CARD-PC-001'})
assert standalone_login['client_token'] and standalone_login['mode'] == 'card', standalone_login
card_cloud = client('/api/client/cloud-vars?app_key=demo-app', headers={'Authorization':'Bearer '+standalone_login['client_token']})
assert card_cloud['data']['notice_text_'+suffix] == '已更新', card_cloud
for row in rows:
    urllib.request.urlopen(urllib.request.Request(BASE+f"/api/admin/cloud-vars/{row['id']}", method='DELETE', headers=auth_header), timeout=10).read()
remaining_cloud = req('/api/admin/cloud-vars?keyword='+suffix, headers=auth_header)['data']
assert not remaining_cloud, remaining_cloud
standalone_hb = client('/api/client/heartbeat','POST',{'machineCode':'CARD-PC-001','clientVersion':'0.1.0'},{'Authorization':'Bearer '+standalone_login['client_token']})
assert standalone_hb.get('ok') is True, standalone_hb
card_devices = req('/api/admin/devices?online=1&keyword=CARD-PC-001', headers=auth_header)['data']
assert card_devices and card_devices[0].get('auth_mode') == 'card' and card_devices[0].get('username') == standalone_card, card_devices
card_unbind = client('/api/client/card-unbind','POST',{'appKey':'demo-app','cardKey':standalone_card})
assert card_unbind.get('ok') is True and card_unbind.get('mode') == 'card', card_unbind
assert int(card_unbind.get('unbind_used', 0)) == 1 and card_unbind.get('deducted') is False, card_unbind
card_after_first_unbind = req('/api/admin/cards?keyword='+standalone_card, headers=auth_header)['data'][0]
assert int(card_after_first_unbind.get('unbind_used', 0)) == 1 and int(card_after_first_unbind.get('expire_hours', 0)) == 2, card_after_first_unbind
standalone_relogin = client('/api/client/card-login','POST',{'appKey':'demo-app','cardKey':standalone_card,'machineCode':'CARD-PC-002'})
assert standalone_relogin['client_token'] and standalone_relogin['machine_code'] == 'CARD-PC-002', standalone_relogin
card_second_unbind = client('/api/client/card-unbind','POST',{'appKey':'demo-app','cardKey':standalone_card})
assert card_second_unbind.get('ok') is True and int(card_second_unbind.get('unbind_used', 0)) == 2 and card_second_unbind.get('deducted') is True, card_second_unbind
card_after_second_unbind = req('/api/admin/cards?keyword='+standalone_card, headers=auth_header)['data'][0]
assert int(card_after_second_unbind.get('unbind_used', 0)) == 2 and int(card_after_second_unbind.get('expire_hours', 0)) == 1, card_after_second_unbind
standalone_relogin2 = client('/api/client/card-login','POST',{'appKey':'demo-app','cardKey':standalone_card,'machineCode':'CARD-PC-003'})
assert standalone_relogin2['client_token'] and standalone_relogin2['machine_code'] == 'CARD-PC-003', standalone_relogin2
card_third_unbind = client('/api/client/card-unbind','POST',{'appKey':'demo-app','cardKey':standalone_card}, expect_error=True)
assert card_third_unbind.get('ok') is False and '解绑次数' in card_third_unbind.get('message',''), card_third_unbind
second_machine = client('/api/client/login','POST',{'appKey':'demo-app','username':user,'password':'123456','machineCode':'PC-002'}, expect_error=True)
assert second_machine.get('ok') is False, 'max device limit should reject second machine login'
hb = client('/api/client/heartbeat','POST',{'machineCode':'PC-001','clientVersion':'0.1.0'},{'Authorization':'Bearer '+client_login['client_token']})
assert hb['heartbeat_interval'] == 45 and hb['heartbeat_timeout'] == 135, hb
print('[5b] device list/status and card filters/export')
devices = req('/api/admin/devices?keyword=PC-001', headers=auth_header)['data']
assert devices, 'device not found'
req(f"/api/admin/devices/{devices[0]['id']}/status", 'PUT', {'status':'banned'}, auth_header)
banned_hb = client('/api/client/heartbeat','POST',{'machineCode':'PC-001','clientVersion':'0.1.1'},{'Authorization':'Bearer '+client_login['client_token']}, expect_error=True)
assert banned_hb.get('ok') is False, 'banned device heartbeat should fail'
req(f"/api/admin/devices/{devices[0]['id']}/status", 'PUT', {'status':'active'}, auth_header)
wrong_machine_hb = client('/api/client/heartbeat','POST',{'machineCode':'PC-OTHER','clientVersion':'0.1.1'},{'Authorization':'Bearer '+client_login['client_token']}, expect_error=True)
assert wrong_machine_hb.get('ok') is False, 'token machine mismatch should fail'
filtered = req('/api/admin/cards?status=used&keyword='+card, headers=auth_header)['data']
assert any(x['card_key'] == card for x in filtered), 'used card filter failed'
export_body = urllib.request.urlopen(urllib.request.Request(BASE+'/api/admin/cards/export?keyword='+card, headers=auth_header), timeout=10).read().decode()
assert card in export_body, 'card export missing key'
print('[5c] user management edit/password/unbind/devices')
users = req('/api/admin/users?keyword='+user, headers=auth_header)['data']
assert users, 'user not found'
assert users[0].get('machine_code') == 'PC-001', users[0]
uid = users[0]['id']
wrong_password = client('/api/client/login','POST',{'appKey':'demo-app','username':user,'password':'wrong-password','machineCode':'PC-001'}, expect_error=True)
assert wrong_password.get('message') == '账号或密码错误', wrong_password
req(f'/api/admin/users/{uid}', 'PUT', {'status':'disabled','expireAt':'2027-01-01 00:00:00','machineCode':'PC-001','maxDevices':2}, auth_header)
client_login_disabled = client('/api/client/login','POST',{'appKey':'demo-app','username':user,'password':'123456','machineCode':'PC-001'}, expect_error=True)
assert client_login_disabled.get('message') == '账号已被禁用', client_login_disabled
req(f'/api/admin/users/{uid}', 'PUT', {'status':'active','expireAt':'2027-01-01 00:00:00','machineCode':'PC-001','maxDevices':2}, auth_header)
req(f'/api/admin/users/{uid}/password', 'PUT', {'password':'654321'}, auth_header)
fresh_login = client('/api/client/login','POST',{'appKey':'demo-app','username':user,'password':'654321','machineCode':'PC-001'})
user_devices = req(f'/api/admin/users/{uid}/devices', headers=auth_header)['data']
assert user_devices, 'user devices empty'
client_unbind = client('/api/client/unbind','POST',{'appKey':'demo-app','machineCode':'PC-001'},{'Authorization':'Bearer '+fresh_login['client_token']})
assert client_unbind.get('ok') is True and client_unbind.get('message') == '解绑成功，请重新登录绑定新机器', client_unbind
user_devices_after_client_unbind = req(f'/api/admin/users/{uid}/devices', headers=auth_header)['data']
assert any(d.get('machine_code') == 'PC-001' and d.get('status') == 'active' for d in user_devices_after_client_unbind), user_devices_after_client_unbind
users_after = req('/api/admin/users?keyword='+user, headers=auth_header)['data']
assert users_after[0].get('machine_code') in ('', None), 'machine code not cleared'
assert int(users_after[0]['unbind_used']) == 1 and int(users_after[0]['free_unbinds']) == 1 and int(users_after[0]['max_unbinds']) == 2, users_after[0]
assert str(users_after[0]['expire_at']).startswith('2027-01-01'), users_after[0]
assert client_unbind.get('deducted') is False, client_unbind
client('/api/client/login','POST',{'appKey':'demo-app','username':user,'password':'654321','machineCode':'PC-NEW'})
account_unbind = client('/api/client/account-unbind','POST',{'appKey':'demo-app','username':user,'password':'654321'})
assert account_unbind.get('ok') is True and int(account_unbind.get('unbind_used', 0)) == 2, account_unbind
assert account_unbind.get('deducted') is True, account_unbind
users_after_account_unbind = req('/api/admin/users?keyword='+user, headers=auth_header)['data']
assert users_after_account_unbind[0].get('machine_code') in ('', None), users_after_account_unbind[0]
assert not str(users_after_account_unbind[0]['expire_at']).startswith('2027-01-01'), 'second unbind should deduct time'
client('/api/client/login','POST',{'appKey':'demo-app','username':user,'password':'654321','machineCode':'PC-NEW2'})
third_unbind = client('/api/client/account-unbind','POST',{'appKey':'demo-app','username':user,'password':'654321'}, expect_error=True)
assert third_unbind.get('ok') is False, 'max unbinds should reject third account/password unbind'
print('[6] agent create/login/scopes')
ts = str(int(time.time()))
agent_user = 'agent' + ts
agent_pass = 'agentpass123'
agent = req('/api/admin/agents','POST',{'username':agent_user,'password':agent_pass,'balance':10,'remark':'verify agent','appIds':[1],'cardTypeIds':[1]},auth_header)
agent_id = agent['id']
agent_login = req('/api/agent/login','POST',{'username':agent_user,'password':agent_pass})
agent_header = {'Authorization':'Bearer '+agent_login['token']}
req('/api/agent/me', headers=agent_header)
req('/api/agent/scopes', headers=agent_header)
print('[7] agent insufficient balance blocked')
bad = req('/api/agent/cards/generate','POST',{'appId':1,'cardTypeId':1,'count':1},agent_header,expect_error=True)
assert bad.get('ok') is False, bad
print('[8] admin add balance then agent issue card')
req(f'/api/admin/agents/{agent_id}/balance','POST',{'type':'add','amount':200,'remark':'verify add'},auth_header)
issued = req('/api/agent/cards/generate','POST',{'appId':1,'cardTypeId':1,'count':5},agent_header)
assert issued['data'], issued
cards = req('/api/agent/cards', headers=agent_header)['data']
assert any(x['card_key'] == issued['data'][0] for x in cards), 'issued card not in agent card list'
sales = req('/api/admin/sales?agent_id='+str(agent_id)+'&card_type_id=1&time_field=created_at&start_date=2026-01-01&end_date=2099-12-31', headers=auth_header)['data']
assert sales['summary']['cards'] >= 2 and sales['summary']['amount'] > 0, sales
agent_sales = req('/api/agent/sales?card_type_id=1&time_field=created_at&start_date=2026-01-01&end_date=2099-12-31', headers=agent_header)['data']
assert agent_sales['summary']['cards'] >= 2 and agent_sales['by_type'], agent_sales
empty_used_sales = req('/api/admin/sales?agent_id='+str(agent_id)+'&card_type_id=1&time_field=used_at&start_date=2026-01-01&end_date=2099-12-31', headers=auth_header)['data']
assert empty_used_sales['summary']['cards'] == 0, empty_used_sales
sales_disabled_card = issued['data'][0]
sales_disabled_rows = req('/api/admin/cards?keyword='+sales_disabled_card, headers=auth_header)['data']
sales_disabled_id = next(x['id'] for x in sales_disabled_rows if x['card_key'] == sales_disabled_card)
req(f"/api/admin/cards/{sales_disabled_id}/disable", 'PUT', {}, auth_header)
sales_without_disabled = req('/api/admin/sales?agent_id='+str(agent_id)+'&card_type_id=1', headers=auth_header)['data']
sales_with_disabled = req('/api/admin/sales?agent_id='+str(agent_id)+'&card_type_id=1&include_disabled=1', headers=auth_header)['data']
assert sales_with_disabled['summary']['cards'] >= sales_without_disabled['summary']['cards'], (sales_without_disabled, sales_with_disabled)
stats = req('/api/agent/stats', headers=agent_header)['data']
assert stats['cards'] >= 1, 'agent stats cards count failed'
logs = req('/api/agent/balance-logs', headers=agent_header)['data']
assert logs, 'agent balance logs empty'
print('[8b] disabled/deleted agent card policy')
agent_card_blocked = issued['data'][1]
agent_card_deleted = issued['data'][2]
agent_account_card = issued['data'][3]
agent_standalone_card = issued['data'][4]
agent_user_client = 'agentclient' + ts
client('/api/client/register','POST',{'appKey':'demo-app','username':agent_user_client,'password':'123456','cardKey':agent_account_card,'machineCode':'AGENT-USER-PC'})
agent_client_login = client('/api/client/login','POST',{'appKey':'demo-app','username':agent_user_client,'password':'123456','machineCode':'AGENT-USER-PC'})
agent_standalone_login = client('/api/client/card-login','POST',{'appKey':'demo-app','cardKey':agent_standalone_card,'machineCode':'AGENT-CARD-PC'})
pre_disable_account_hb = client('/api/client/heartbeat','POST',{'machineCode':'AGENT-USER-PC','clientVersion':'0.1.0'},{'Authorization':'Bearer '+agent_client_login['client_token']})
assert pre_disable_account_hb.get('ok') is True, pre_disable_account_hb
pre_disable_card_hb = client('/api/client/heartbeat','POST',{'machineCode':'AGENT-CARD-PC','clientVersion':'0.1.0'},{'Authorization':'Bearer '+agent_standalone_login['client_token']})
assert pre_disable_card_hb.get('ok') is True, pre_disable_card_hb
req(f'/api/admin/agents/{agent_id}/status', 'PUT', {'status':'disabled'}, auth_header)
blocked = client('/api/client/card-login','POST',{'appKey':'demo-app','cardKey':agent_card_blocked,'machineCode':'AGENT-BLOCKED-PC'}, expect_error=True)
assert blocked.get('ok') is False, 'disabled agent unused card should not recharge/login'
blocked_account_hb = client('/api/client/heartbeat','POST',{'machineCode':'AGENT-USER-PC','clientVersion':'0.1.1'},{'Authorization':'Bearer '+agent_client_login['client_token']}, expect_error=True)
assert blocked_account_hb.get('message') == '授权来源代理已被禁用', blocked_account_hb
blocked_card_hb = client('/api/client/heartbeat','POST',{'machineCode':'AGENT-CARD-PC','clientVersion':'0.1.1'},{'Authorization':'Bearer '+agent_standalone_login['client_token']}, expect_error=True)
assert blocked_card_hb.get('ok') is False, blocked_card_hb
req(f'/api/admin/agents/{agent_id}/status', 'PUT', {'status':'active'}, auth_header)
req(f'/api/admin/agents/{agent_id}', 'DELETE', {}, auth_header)
deleted_cards = req('/api/admin/cards?keyword='+agent_card_deleted, headers=auth_header)['data']
assert not any(x['card_key'] == agent_card_deleted for x in deleted_cards), 'deleting agent should delete its cards'
print('agent_card='+agent_card_blocked)
print('OK verify passed')
