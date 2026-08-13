/* ============================================================
   obsidian-bridge.js — NR_OB 桥接 C（桌面版 Wails 壳包装）
   ------------------------------------------------------------
   A+C 架构：通用核心 A（笔记主页.html）通过 window.NR_OB 访问环境能力；
   C = 本包装——把 wails 绑定的 Go 桥接（window.go.main.Bridge.*）
   包装成与 Obsidian 版（B 桥接）等价的 NR_OB 接口。
   A 侧 6 处调用点零改动即复用桌面桥接。

   方法名与 Obsidian 版契约一致：
     isObsidian() / vaultPath() / saveNote(md,meta) / openNote(id,title,path) /
     deleteNote(path) / recordQuestion(q) / getSettings()
   若 wails 绑定名带包前缀不同，按实际调整下方 go 引用。
   ============================================================ */
(function(){
  'use strict';
  // wails 运行时绑定：window.go.<包名小写>.<类型名>（main 包 + Bridge 类型）
  var goBridge = window.go && (window.go.bridge || (window.go.main && window.go.main.Bridge));
  if(!goBridge){
    console.warn('[NR_OB] Go 桥接未绑定（window.go.main.Bridge 缺失），NR_OB 不可用');
  }
  window.NR_OB = {
    // 桌面宿主环境恒真（Go 侧 IsObsidian 返回 true）
    isObsidian: function(){ return goBridge ? goBridge.IsObsidian() : Promise.resolve(false); },
    vaultPath: function(){ return goBridge ? goBridge.VaultPath() : Promise.reject(new Error('[NR_OB] 桥接未绑定')); },
    saveNote: function(md, meta){ return goBridge ? goBridge.SaveNote(md, JSON.stringify(meta||{})) : Promise.reject(new Error('[NR_OB] 桥接未绑定')); },
    openNote: function(id, title, path){ return goBridge ? goBridge.OpenNote(id||'', title||'', path||'') : Promise.reject(new Error('[NR_OB] 桥接未绑定')); },
    // 删除笔记：Go 侧 os.Remove（同款路径校验防穿越）；文件不存在 resolve false
    deleteNote: function(path){ return goBridge ? goBridge.DeleteNote(path||'') : Promise.reject(new Error('[NR_OB] 桥接未绑定')); },
    // 外链打开：Go 侧系统默认浏览器（WebView2 内 window.open 无多窗口支持）
    openUrl: function(url){ return goBridge ? goBridge.OpenURL(url||'') : Promise.reject(new Error('[NR_OB] 桥接未绑定')); },
    // 标题栏颜色跟随主题：DWM 动态着色（bgHex/textHex 为 #RRGGBB）；无此能力的环境静默
    setTitleBarColor: function(bgHex, textHex){ return goBridge ? goBridge.SetTitleBarColor(bgHex||'', textHex||'') : Promise.reject(new Error('[NR_OB] 桥接未绑定')); },
    // 标题栏深浅模式：DWM immersive dark mode（dark=深色标题栏浅色文字）；无此能力的环境静默
    setTitleBarMode: function(dark){ return goBridge ? goBridge.SetTitleBarMode(!!dark) : Promise.reject(new Error('[NR_OB] 桥接未绑定')); },
    recordQuestion: function(q){ return goBridge ? goBridge.RecordQuestion(JSON.stringify(q||{})) : Promise.reject(new Error('[NR_OB] 桥接未绑定')); },
    getSettings: function(){ return goBridge ? goBridge.GetSettings().then(function(s){ try{ return JSON.parse(s); }catch(e){ return {}; } }) : Promise.reject(new Error('[NR_OB] 桥接未绑定')); },
    // 网络抓取：C 桥接代发（Go 无浏览器 CORS 限制），返回 fetch Response 兼容子集
    // （A 侧直抓代码用 res.ok / res.status / res.text() / res.json()）
    fetch: function(url){
      return goBridge ? goBridge.Fetch(url).then(function(r){
        var body = r.body || '';
        var status = parseInt(r.status, 10) || 0;
        return {
          ok: status >= 200 && status < 300,
          status: status,
          text: function(){ return Promise.resolve(body); },
          json: function(){ try{ return Promise.resolve(JSON.parse(body)); }catch(e){ return Promise.reject(e); } }
        };
      }) : Promise.reject(new Error('[NR_OB] 桥接未绑定'));
    }
  };
  console.log('[NR_OB] 桌面桥接就绪：'+Object.keys(window.NR_OB).join(','));
})();
