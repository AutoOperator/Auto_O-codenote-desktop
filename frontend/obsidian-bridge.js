/* ============================================================
   obsidian-bridge.js — NR_OB 桥接 C（桌面版 Wails 壳包装）
   ------------------------------------------------------------
   A+C 架构：通用核心 A（笔记主页.html）通过 window.NR_OB 访问环境能力；
   C = 本包装——把 wails 绑定的 Go 桥接（window.go.main.Bridge.*）
   包装成与 Obsidian 版（B 桥接）等价的 NR_OB 接口。
   A 侧 6 处调用点零改动即复用桌面桥接。

   方法名与 Obsidian 版契约一致：
     isObsidian() / vaultPath() / saveNote(md,meta) /
     openNote(id,title,path) / recordQuestion(q) / getSettings()
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
    recordQuestion: function(q){ return goBridge ? goBridge.RecordQuestion(JSON.stringify(q||{})) : Promise.reject(new Error('[NR_OB] 桥接未绑定')); },
    getSettings: function(){ return goBridge ? goBridge.GetSettings().then(function(s){ try{ return JSON.parse(s); }catch(e){ return {}; } }) : Promise.reject(new Error('[NR_OB] 桥接未绑定')); }
  };
  console.log('[NR_OB] 桌面桥接就绪：'+Object.keys(window.NR_OB).join(','));
})();
