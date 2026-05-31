<template>
  <div class="api-doc-page">
    <section class="api-hero-card">
      <div>
        <span class="api-pill">CLIENT SECURITY</span>
        <h1>客户端接口 / 安全协议</h1>
        <p>客户端所有接口统一走 <b>/api/client/*</b>，必须使用软件专属 <b>client_secret</b> 做 HMAC-SHA256 签名；账号注册、登录、充值、解绑、心跳都按同一套规则对接。</p>
      </div>
      <div class="api-flow">
        <span>取软件信息</span><i></i><span>签名请求</span><i></i><span>登录授权</span><i></i><span>心跳/解绑</span>
      </div>
    </section>

    <n-grid :cols="3" :x-gap="16" :y-gap="16" responsive="screen">
      <n-gi v-for="card in summaryCards" :key="card.title">
        <n-card class="api-info-card">
          <small>{{ card.kicker }}</small>
          <b>{{ card.title }}</b>
          <p>{{ card.text }}</p>
        </n-card>
      </n-gi>
    </n-grid>

    <n-card class="api-panel" title="一、所有客户端接口必带 Header" style="margin-top:16px">
      <div class="header-grid">
        <div v-for="h in headers" :key="h.name"><b>{{ h.name }}</b><span>{{ h.desc }}</span></div>
      </div>
      <n-alert type="warning" style="margin-top:14px">
        X-Signature 必须是 64 位小写 hex 文本。不要把 HMAC 原始字节集直接放进请求头；易语言里要先把字节集转十六进制文本。
      </n-alert>
    </n-card>

    <n-card class="api-panel" title="二、签名原文拼接规则" style="margin-top:16px">
      <div class="signature-layout">
        <pre class="code-block">{{ canonical }}</pre>
        <div class="api-tips compact">
          <p><b>METHOD</b><span>大写请求方法，例如 GET / POST。</span></p>
          <p><b>PATH_WITH_QUERY</b><span>只签路径和 query，例如 /api/client/app-info?app_key=demo-app，不带 http://域名:端口。</span></p>
          <p><b>TIMESTAMP</b><span>当前 Unix 秒，10 位，不要用 13 位毫秒。</span></p>
          <p><b>NONCE</b><span>每次请求唯一；同一个 nonce 重复会返回请求已被使用。</span></p>
          <p><b>BODY_JSON</b><span>POST 必须和实际提交 JSON 完全一致；GET 空 body 最后一行仍保留换行。</span></p>
        </div>
      </div>
    </n-card>

    <section class="endpoint-grid">
      <article v-for="ep in mainEndpoints" :key="ep.path" class="endpoint-card">
        <div class="endpoint-head">
          <n-tag :type="ep.method==='GET'?'info':'success'" round>{{ ep.method }}</n-tag>
          <code>{{ ep.path }}</code>
        </div>
        <h3>{{ ep.title }}</h3>
        <p>{{ ep.desc }}</p>
        <div class="param-box">
          <b>请求参数</b>
          <span v-for="p in ep.params" :key="p">{{ p }}</span>
        </div>
        <pre>{{ ep.body }}</pre>
        <div class="return-box"><b>成功返回</b><span>{{ ep.returns }}</span></div>
      </article>
    </section>

    <section class="endpoint-grid endpoint-grid-single">
      <article v-for="ep in wideEndpoints" :key="ep.path" class="endpoint-card endpoint-card-wide">
        <div class="endpoint-head">
          <n-tag type="success" round>{{ ep.method }}</n-tag>
          <code>{{ ep.path }}</code>
        </div>
        <h3>{{ ep.title }}</h3>
        <p>{{ ep.desc }}</p>
        <div class="param-box">
          <b>请求参数</b>
          <span v-for="p in ep.params" :key="p">{{ p }}</span>
        </div>
        <pre>{{ ep.body }}</pre>
        <div class="return-box"><b>成功返回</b><span>{{ ep.returns }}</span></div>
      </article>
    </section>

    <n-grid :cols="2" :x-gap="16" :y-gap="16" responsive="screen" style="margin-top:16px">
      <n-gi>
        <n-card class="api-panel" title="易语言接入重点">
          <div class="api-tips">
            <p><b>换行符</b><span>签名原文用 字符(10)，不要用可能变成 CRLF 的 #换行符。</span></p>
            <p><b>签名密钥</b><span>HMAC 的 key 是软件里的 client_secret，不是 appKey，也不是卡密。</span></p>
            <p><b>签名结果</b><span>HMAC_SHA256 返回字节集后，要转 64 位小写十六进制，再放 X-Signature。</span></p>
            <p><b>POST JSON</b><span>签名前的 JSON 字符串必须和网页_访问S 实际提交内容一模一样。</span></p>
          </div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card class="api-panel" title="常见错误 message">
          <div class="api-tips">
            <p><b>签名校验失败</b><span>签名原文、密钥、hex、路径或 body 不一致。</span></p>
            <p><b>请求已被使用</b><span>nonce 重复，重新生成随机串。</span></p>
            <p><b>机器码不匹配</b><span>登录 token 或卡密绑定的机器码和提交机器码不同。</span></p>
            <p><b>已达到最大绑定设备数</b><span>需要后台/客户端解绑后再登录新机器。</span></p>
          </div>
        </n-card>
      </n-gi>
    </n-grid>
  </div>
</template>
<script setup>
const summaryCards = [
  {kicker:'签名算法', title:'HMAC-SHA256', text:'所有 /api/client/* 都要签名，签名结果放 X-Signature。'},
  {kicker:'密钥来源', title:'软件 client_secret', text:'在软件管理里复制完整客户端密钥；重置后旧客户端会失效。'},
  {kicker:'授权模式', title:'账号 + 卡密 + 解绑', text:'支持注册激活、账号登录、卡密登录、充值续费、客户端解绑和心跳。'},
]

const headers = [
  {name:'X-App-Key', desc:'软件 App Key，例如 demo-app'},
  {name:'X-Timestamp', desc:'当前 Unix 秒，10 位'},
  {name:'X-Nonce', desc:'每次请求唯一随机串'},
  {name:'X-Signature', desc:'HMAC-SHA256 小写 hex'},
  {name:'Content-Type', desc:'POST 使用 application/json'},
  {name:'Authorization', desc:'解绑/心跳携带 Bearer client_token'},
]

const canonical = `METHOD
PATH_WITH_QUERY
TIMESTAMP
NONCE
BODY_JSON`

const endpoints = [
  {
    method:'GET', path:'/api/client/app-info?app_key=demo-app', title:'软件运行信息',
    desc:'客户端启动时读取软件名称、公告、版本、强更、下载地址和心跳配置。GET 请求 body 为空，但签名原文末尾仍要有最后一个换行。',
    params:['Query app_key：软件标识','Header X-App-Key：同一个软件 AppKey'],
    body:`签名原文示例：
GET
/api/client/app-info?app_key=demo-app
时间戳
随机串
`,
    returns:'data[0].version / min_version / force_update / announcement / heartbeat_interval / heartbeat_timeout',
  },
  {
    method:'GET', path:'/api/client/cloud-vars?app_key=demo-app', title:'云变量 / 云配置',
    desc:'读取软件下发的动态配置、开关、数字参数和 JSON 配置，适合客户端开关功能。',
    params:['Query app_key：软件标识'],
    body:`返回示例：
{"notice_text":"欢迎使用","enable_feature_x":true,"max_thread":8}`,
    returns:'data 是 key/value 对象，bool/number/json 会自动转类型。',
  },
  {
    method:'POST', path:'/api/client/register', title:'账号注册并卡密激活',
    desc:'账号密码模式入口。注册时必须提交卡密和机器码，成功后立即消费卡密、写入机器码、生成设备记录，并返回 client_token。',
    params:['appKey','username','password','cardKey','machineCode'],
    body:`{
  "appKey": "demo-app",
  "username": "u1",
  "password": "123456",
  "cardKey": "LS-XXXX",
  "machineCode": "PC-001"
}`,
    returns:'client_token / user_id / app_id / expire_at / machine_code / heartbeat_interval / heartbeat_timeout',
  },
  {
    method:'POST', path:'/api/client/login', title:'账号密码登录',
    desc:'校验账号、密码、状态、到期时间和机器码绑定。首次机器登录会绑定设备；超过最大设备数会拒绝。',
    params:['appKey','username','password','machineCode'],
    body:`{
  "appKey": "demo-app",
  "username": "u1",
  "password": "123456",
  "machineCode": "PC-001"
}`,
    returns:'client_token / expire_at / max_devices / machine_code / heartbeat_interval / heartbeat_timeout',
  },
  {
    method:'POST', path:'/api/client/recharge', title:'账号卡密充值 / 续费',
    desc:'已有账号使用新卡密续费。未到期时在原到期时间上叠加，已过期则从当前时间开始。',
    params:['appKey','username','cardKey'],
    body:`{
  "appKey": "demo-app",
  "username": "u1",
  "cardKey": "LS-YYYY"
}`,
    returns:'expire_at / max_devices，返回新的到期时间。',
  },
  {
    method:'POST', path:'/api/client/card-login', title:'单独卡密登录',
    desc:'不注册账号，直接用卡密授权。首次登录绑定机器码，后续必须同一卡密 + 同一机器码。',
    params:['appKey','cardKey','machineCode'],
    body:`{
  "appKey": "demo-app",
  "cardKey": "LS-ZZZZ",
  "machineCode": "PC-001"
}`,
    returns:'mode=card / client_token / expire_at / machine_code / heartbeat_interval / heartbeat_timeout',
  },
  {
    method:'POST', path:'/api/client/unbind', title:'客户端账号解绑（token）',
    desc:'账号已登录时使用，需 Authorization。解绑只清空用户机器码、保留历史设备记录，并按卡类解绑规则扣次数/扣时。卡密登录模式请用卡密解绑。',
    params:['Authorization: Bearer client_token','appKey','machineCode'],
    body:`{
  "appKey": "demo-app",
  "machineCode": "PC-001"
}`,
    returns:'message / unbind_used / expire_at / deducted / deduct_hours；成功后需要重新登录绑定新机器。',
  },
  {
    method:'POST', path:'/api/client/account-unbind', title:'账号密码解绑',
    desc:'换机器时旧机器无法登录也能解绑：只要账号密码正确即可清空绑定机器码，不需要 client_token。',
    params:['appKey','username','password'],
    body:`{
  "appKey": "demo-app",
  "username": "u1",
  "password": "123456"
}`,
    returns:'message / unbind_used / expire_at / deducted / deduct_hours；成功后新机器重新登录即可绑定。',
  },
  {
    method:'POST', path:'/api/client/card-unbind', title:'卡密解绑',
    desc:'单独卡密登录模式使用。提交 appKey + cardKey 即可清空卡密绑定机器码，不需要 client_token。',
    params:['appKey','cardKey'],
    body:`{
  "appKey": "demo-app",
  "cardKey": "LS-ZZZZ"
}`,
    returns:'mode=card / message / expire_at；成功后用同一卡密在新机器 card-login。',
  },
  {
    method:'POST', path:'/api/client/heartbeat', title:'客户端心跳',
    desc:'携带登录或卡密登录返回的 client_token，持续校验授权、机器码、设备状态和到期时间。设备被后台禁用/封禁后，下一次心跳会失败。',
    params:['Authorization: Bearer client_token','machineCode','clientVersion'],
    body:`{
  "machineCode": "PC-001",
  "clientVersion": "1.0.0"
}`,
    returns:'server_time / heartbeat_interval / heartbeat_timeout。',
  },
]
const mainEndpoints = endpoints.slice(0, 6)
const wideEndpoints = endpoints.slice(6)
</script>
