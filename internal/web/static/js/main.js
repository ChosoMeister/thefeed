// ===== BACKGROUND IMAGE =====
function _setBg(url) {
  var ca = document.querySelector('.chat-area');
  ca.style.backgroundImage = url ? 'url("' + url + '")' : '';
  ca.style.backgroundSize = url ? 'cover' : '';
  ca.style.backgroundPosition = url ? 'center' : '';
  ca.style.backgroundRepeat = url ? 'no-repeat' : '';
  document.getElementById('messages').style.background = url ? 'transparent' : '';
}
function loadBgImage() {
  // Use cache-busting query to ensure latest image.
  var url = '/api/bg-image?t=' + Date.now();
  fetch(url).then(function (r) {
    if (r.status === 204 || !r.ok) return;
    _setBg('/api/bg-image?t=' + Date.now());
  }).catch(function () { });
}
async function applyBgImage() {
  var inp = document.getElementById('bgImageInput');
  if (!inp.files || !inp.files[0]) return;
  var file = inp.files[0];
  if (file.size > 10 * 1024 * 1024) { showToast('File too large (max 10MB)'); return }
  try {
    // Buffer via FileReader so content:// URIs from Google Photos
    // work too — fetch's stream path fails on those in some WebViews.
    var buf = await new Promise(function (resolve, reject) {
      var fr = new FileReader();
      fr.onload = function () { resolve(fr.result); };
      fr.onerror = function () { reject(fr.error || new Error('read failed')); };
      fr.readAsArrayBuffer(file);
    });
    var r = await fetch('/api/bg-image', {
      method: 'POST',
      headers: { 'Content-Type': file.type || 'application/octet-stream' },
      body: buf
    });
    if (!r.ok) { showToast(await r.text()); return }
    _setBg('/api/bg-image?t=' + Date.now());
    showToast(t('apply'));
  } catch (e) { showToast((e && e.message) || 'failed'); }
}
async function clearBgImage() {
  try { await fetch('/api/bg-image', { method: 'DELETE' }) } catch (e) { }
  _setBg('');
  document.getElementById('bgImageInput').value = '';
  showToast(t('clear_bg'));
}

// ===== EVENTS =====
document.addEventListener('keydown', function (e) {
  if (e.key === 'Enter' && document.activeElement === document.getElementById('sendInput')) { e.preventDefault(); sendMessage() }
  if (e.key === 'Enter' && document.activeElement === document.getElementById('peAddChannelInput')) { e.preventDefault(); addChannelEditor() }
  if (e.key === 'Enter' && document.activeElement === document.getElementById('msgSearchInput')) { e.preventDefault(); msgSearchNext() }
  if (e.key === 'Escape') { closeSettings(); closeProfiles(); closeProfileEditor(); closeScanner(); closeMsgSearch(); closeExportModal(); closeResolversModal(); closeTelemirror() }
});
mobileQuery.addEventListener('change', function () {
  var app = document.getElementById('app');
  if (!mobileQuery.matches) {
    app.classList.remove('chat-open');
  } else if (chatIsOpen) {
    app.classList.add('chat-open');
  }
});

// Handle thefeed:// URI hash import
(function () { var h = location.hash; if (h && h.startsWith('#thefeed://')) { document.getElementById('importUriInput').value = decodeURIComponent(h.substring(1)); openProfiles(); doImportUri() } })();

// ===== AUTO-SCROLL DURING TEXT SELECTION =====
(function () {
  var scrollSpeed = 0, scrollFrame = null, messagesEl = null;
  function startAutoScroll() {
    if (scrollFrame) return;
    function step() {
      if (scrollSpeed === 0 || !messagesEl) { scrollFrame = null; return }
      messagesEl.scrollTop += scrollSpeed;
      scrollFrame = requestAnimationFrame(step);
    }
    scrollFrame = requestAnimationFrame(step);
  }
  function stopAutoScroll() {
    scrollSpeed = 0;
    if (scrollFrame) { cancelAnimationFrame(scrollFrame); scrollFrame = null }
  }
  document.addEventListener('DOMContentLoaded', function () {
    messagesEl = document.getElementById('messages');
    if (!messagesEl) return;
    var edgeZone = 40;
    function handleMove(clientY) {
      var sel = window.getSelection();
      if (!sel || sel.isCollapsed) return;
      var rect = messagesEl.getBoundingClientRect();
      if (clientY < rect.top + edgeZone) {
        scrollSpeed = -Math.max(2, (edgeZone - (clientY - rect.top)) / 3);
        startAutoScroll();
      } else if (clientY > rect.bottom - edgeZone) {
        scrollSpeed = Math.max(2, (edgeZone - (rect.bottom - clientY)) / 3);
        startAutoScroll();
      } else { stopAutoScroll() }
    }
    messagesEl.addEventListener('touchmove', function (e) { if (e.touches[0]) handleMove(e.touches[0].clientY) });
    messagesEl.addEventListener('touchend', stopAutoScroll);
    messagesEl.addEventListener('touchcancel', stopAutoScroll);
    messagesEl.addEventListener('mousemove', function (e) { if (e.buttons === 1) handleMove(e.clientY) });
    document.addEventListener('mouseup', stopAutoScroll);
  });
})();

// Close modals (bottom sheets) when clicking outside
(function () {
  var modalMap = {
    settingsModal: function () { closeSettings() },
    profilesModal: function () { closeProfiles() },
    profileEditorModal: function () { closeProfileEditor && closeProfileEditor() },
    shareProfileModal: function () { closeShareModal() },
    exportModal: function () { closeExportModal() },
    resolversModal: function () { closeResolversModal() },
    scannerModal: function () { closeScanner() },
    savedResolversModal: function () { savedResolversSkip && savedResolversSkip() },
  };
  document.addEventListener('click', function (e) {
    var overlay = e.target;
    if (!overlay.classList.contains('modal-overlay') || !overlay.classList.contains('active')) return;
    // Only close if user clicked directly on the overlay backdrop, not the modal content
    if (e.target !== overlay) return;
    var fn = modalMap[overlay.id];
    if (fn) fn();
  });
})();

init();
