/* ==========================================================================
   byok — Official Website Script
   ========================================================================== */
'use strict';

/* --- Terminal Typing Animation --- */
(function () {
  const TYPING_TEXT = 'byok launch copilot';
  const TYPING_SPEED = 80;   // ms per character
  const HOLD_DELAY  = 2000;  // ms to hold after completing
  const ERASE_SPEED = 40;    // ms per character when erasing
  const RESTART_DELAY = 500; // ms before re-typing

  function startTyping() {
    const el = document.getElementById('typing-text');
    if (!el) return;

    let i = 0;
    let erasing = false;

    function tick() {
      if (!erasing) {
        if (i < TYPING_TEXT.length) {
          el.textContent = TYPING_TEXT.slice(0, i + 1);
          i++;
          setTimeout(tick, TYPING_SPEED);
        } else {
          setTimeout(() => {
            erasing = true;
            tick();
          }, HOLD_DELAY);
        }
      } else {
        if (i > 0) {
          i--;
          el.textContent = TYPING_TEXT.slice(0, i);
          setTimeout(tick, ERASE_SPEED);
        } else {
          erasing = false;
          setTimeout(tick, RESTART_DELAY);
        }
      }
    }

    tick();
  }

  document.addEventListener('DOMContentLoaded', startTyping);
})();

/* --- OS Tab Switching --- */
(function () {
  document.addEventListener('DOMContentLoaded', function () {
    const tabBtns = document.querySelectorAll('.os-tab-btn');
    const tabPanels = document.querySelectorAll('.os-tab-panel');

    tabBtns.forEach(function (btn) {
      btn.addEventListener('click', function () {
        const os = btn.getAttribute('data-os');

        tabBtns.forEach(function (b) { b.classList.remove('active'); });
        tabPanels.forEach(function (p) { p.classList.remove('active'); });

        btn.classList.add('active');
        document.querySelector('.os-tab-panel[data-os="' + os + '"]').classList.add('active');
      });
    });
  });
})();

/* --- Latest Release Install Commands --- */
(function () {
  const RELEASE_API = 'https://api.github.com/repos/IISI-2209026/LlmByok/releases/latest';
  const RELEASE_DOWNLOAD_BASE = 'https://github.com/IISI-2209026/LlmByok/releases/download';

  function hasAsset(release, name) {
    return Array.isArray(release.assets) && release.assets.some(function (asset) {
      return asset && asset.name === name;
    });
  }

  function buildCommands(release) {
    if (!release || !release.tag_name) return null;

    const version = release.tag_name.replace(/^v/, '');
    const tag = release.tag_name;
    const linuxAsset = 'byok-' + version + '-linux-amd64.tar.gz';
    const macIntelAsset = 'byok-' + version + '-darwin-amd64.tar.gz';
    const macAppleAsset = 'byok-' + version + '-darwin-arm64.tar.gz';
    const windowsAsset = 'byok-' + version + '-windows-amd64.zip';

    if (
      !hasAsset(release, linuxAsset) ||
      !hasAsset(release, macIntelAsset) ||
      !hasAsset(release, macAppleAsset) ||
      !hasAsset(release, windowsAsset)
    ) {
      return null;
    }

    const base = RELEASE_DOWNLOAD_BASE + '/' + tag;

    return {
      linux: 'curl -L "' + base + '/' + linuxAsset + '" -o byok.tar.gz\n' +
        'tar -xzf byok.tar.gz\n' +
        'sudo install -m 755 byok /usr/local/bin/byok',
      macos: 'curl -L "' + base + '/byok-' + version + '-darwin-$(uname -m | sed \'s/x86_64/amd64/\').tar.gz" -o byok.tar.gz\n' +
        'tar -xzf byok.tar.gz\n' +
        'sudo install -m 755 byok /usr/local/bin/byok',
      windows: 'Invoke-WebRequest -Uri "' + base + '/' + windowsAsset + '" -OutFile "$env:TEMP\\byok.zip"\n' +
        'Expand-Archive -Force "$env:TEMP\\byok.zip" -DestinationPath "$env:TEMP\\byok"\n' +
        'New-Item -ItemType Directory -Force "$env:LOCALAPPDATA\\byok" | Out-Null; Copy-Item "$env:TEMP\\byok\\byok.exe" "$env:LOCALAPPDATA\\byok\\byok.exe" -Force'
    };
  }

  function applyCommands(commands) {
    Object.keys(commands).forEach(function (os) {
      const el = document.querySelector('[data-install-command="' + os + '"]');
      if (el) el.textContent = commands[os];
    });
  }

  document.addEventListener('DOMContentLoaded', function () {
    if (!window.fetch) return;

    fetch(RELEASE_API, { headers: { Accept: 'application/vnd.github+json' } })
      .then(function (response) {
        if (!response.ok) throw new Error('release lookup failed');
        return response.json();
      })
      .then(function (release) {
        const commands = buildCommands(release);
        if (commands) applyCommands(commands);
      })
      .catch(function () {
        // Keep the verified static fallback commands in the HTML.
      });
  });
})();

/* --- Copy to Clipboard --- */
(function () {
  function copyText(text, btn) {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(
        function () { showCopied(btn); },
        function () { fallbackCopy(text, btn); }
      );
    } else {
      fallbackCopy(text, btn);
    }
  }

  function fallbackCopy(text, btn) {
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    try {
      document.execCommand('copy');
      showCopied(btn);
    } catch (e) {
      btn.innerHTML = '<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/></svg>';
      setTimeout(restoreIcon, 2000, btn);
    }
    document.body.removeChild(textarea);
  }

  function showCopied(btn) {
    btn.innerHTML = '<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/></svg>';
    btn.classList.add('copied');
    setTimeout(restoreIcon, 2000, btn);
  }

  function restoreIcon(btn) {
    btn.innerHTML = '<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path d="M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z"/></svg>';
    btn.classList.remove('copied');
  }

  document.addEventListener('DOMContentLoaded', function () {
    const buttons = document.querySelectorAll('.copy-btn');
    buttons.forEach(function (btn) {
      btn.addEventListener('click', function () {
        const block = btn.closest('.code-block');
        if (!block) return;
        const codeEl = block.querySelector('code') || block.querySelector('.code-text');
        const text = codeEl ? codeEl.textContent.trim() : '';
        copyText(text, btn);
      });
    });
  });
})();
