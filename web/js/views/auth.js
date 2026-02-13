// ============================================
// 认证页面 (快速登录 - 仅需用户名)
// ============================================

import { store } from '../store.js';
import { api } from '../api.js';
import { toast } from '../utils.js';
import { navigate } from '../app.js';

export function renderAuth() {
  return `
    <div class="auth-page">
      <div class="auth-container">
        <div class="auth-logo">
          <h1>🩸 血染钟楼</h1>
          <div class="subtitle">Blood on the Clocktower</div>
        </div>
        <div class="auth-card">
          <h2 style="text-align:center; margin-bottom: 1.5rem; color: var(--text-primary);">设置你的名字</h2>
          ${window.__BOTC_AS ? `<div style="text-align:center; margin-bottom:0.5rem; color:var(--text-secondary); font-size:0.85rem;">🔹 小号 #${window.__BOTC_AS}</div>` : ''}
          <form class="auth-form" id="auth-form">
            <div class="form-group">
              <label>用户名</label>
              <input class="input" type="text" id="auth-name" placeholder="请输入你的名字" required autocomplete="username" maxlength="20" minlength="1">
            </div>
            <button type="submit" class="btn btn-primary btn-lg" id="auth-submit">进入游戏</button>
          </form>
        </div>
      </div>
    </div>
  `;
}

export function mountAuth() {
  // Auto-focus the name input
  const nameInput = document.getElementById('auth-name');
  if (nameInput) nameInput.focus();

  // Form submit
  document.getElementById('auth-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const name = document.getElementById('auth-name').value.trim();

    if (!name) {
      toast('请输入你的名字', 'warning');
      return;
    }

    const btn = document.getElementById('auth-submit');
    btn.disabled = true;
    btn.textContent = '进入中...';

    try {
      const result = await api.quickLogin(name);
      console.log('[Auth] Quick login response:', JSON.stringify(result));

      if (!result.token) {
        throw new Error('服务器未返回 token');
      }

      store.setAuth(result.token, result.user_id, name);
      toast(`欢迎，${name}！`, 'success');
      navigate('home');
    } catch (err) {
      toast(err.message || '进入失败', 'error');
      btn.disabled = false;
      btn.textContent = '进入游戏';
    }
  });
}
