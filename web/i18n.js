(() => {
  'use strict';

  const STORAGE_KEY = 'gba-media-maker-language';
  const SUPPORTED = ['en', 'uk'];
  const catalogs = new Map();
  let language = 'en';
  let applying = false;

  function normalizeLanguage(value) {
    const code = String(value || '').trim().toLowerCase().split(/[-_]/)[0];
    return SUPPORTED.includes(code) ? code : 'en';
  }

  function detectLanguage() {
    const nativeSaved = document.querySelector('meta[name=\"gbavm-language\"]')?.content;
    if (nativeSaved) return normalizeLanguage(nativeSaved);
    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (saved) return normalizeLanguage(saved);
    } catch (_) {}
    return normalizeLanguage(navigator.language || 'en');
  }

  function escapeRegex(value) {
    return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  }

  function compileTemplate(template) {
    const names = [];
    let source = '';
    let last = 0;
    for (const match of template.matchAll(/\{([a-zA-Z0-9_]+)\}/g)) {
      source += escapeRegex(template.slice(last, match.index));
      source += '(.+?)';
      names.push(match[1]);
      last = match.index + match[0].length;
    }
    source += escapeRegex(template.slice(last));
    return { regex: new RegExp(`^${source}$`, 'u'), names };
  }

  function fill(template, params = {}) {
    return String(template).replace(/\{([a-zA-Z0-9_]+)\}/g, (_, name) =>
      Object.prototype.hasOwnProperty.call(params, name) ? String(params[name]) : `{${name}}`
    );
  }

  function messagesFor(code) {
    return catalogs.get(code)?.messages || {};
  }

  function translateFromEnglish(source, targetCode) {
    const target = messagesFor(targetCode);
    if (Object.prototype.hasOwnProperty.call(target, source)) return target[source];

    for (const [template, translated] of Object.entries(target)) {
      if (!template.includes('{')) continue;
      const { regex, names } = compileTemplate(template);
      const match = source.match(regex);
      if (!match) continue;
      const params = {};
      names.forEach((name, index) => { params[name] = match[index + 1]; });
      return fill(translated, params);
    }
    return source;
  }

  function canonicalEnglish(value) {
    const source = String(value);
    const uk = messagesFor('uk');
    for (const [english, translated] of Object.entries(uk)) {
      if (!english.includes('{')) {
        if (source === translated) return english;
        continue;
      }
      const { regex, names } = compileTemplate(translated);
      const match = source.match(regex);
      if (!match) continue;
      const params = {};
      names.forEach((name, index) => { params[name] = match[index + 1]; });
      return fill(english, params);
    }
    return source;
  }

  function translateMessage(value, targetCode = language) {
    if (value == null) return value;
    const original = String(value);
    if (!original.trim()) return original;
    const leading = original.match(/^\s*/u)?.[0] || '';
    const trailing = original.match(/\s*$/u)?.[0] || '';
    const core = original.slice(leading.length, original.length - trailing.length);
    const english = canonicalEnglish(core);
    const translated = targetCode === 'en' ? english : translateFromEnglish(english, targetCode);
    return leading + translated + trailing;
  }

  function t(key, params = {}) {
    const english = fill(messagesFor('en')[key] ?? key, params);
    return language === 'en' ? english : translateFromEnglish(english, language);
  }

  function translateNode(root) {
    if (!root || applying) return;
    applying = true;
    try {
      if (root.nodeType === Node.TEXT_NODE) {
        const next = translateMessage(root.nodeValue);
        if (next !== root.nodeValue) root.nodeValue = next;
        return;
      }
      if (root.nodeType !== Node.ELEMENT_NODE && root.nodeType !== Node.DOCUMENT_NODE && root.nodeType !== Node.DOCUMENT_FRAGMENT_NODE) return;

      const elements = [];
      if (root.nodeType === Node.ELEMENT_NODE) elements.push(root);
      if (root.querySelectorAll) elements.push(...root.querySelectorAll('*'));
      for (const element of elements) {
        if (element.matches?.('script,style,code,pre,[data-i18n-ignore]')) continue;
        for (const attr of ['aria-label', 'title', 'placeholder']) {
          if (!element.hasAttribute?.(attr)) continue;
          const value = element.getAttribute(attr);
          const next = translateMessage(value);
          if (next !== value) element.setAttribute(attr, next);
        }
        for (const child of element.childNodes || []) {
          if (child.nodeType !== Node.TEXT_NODE) continue;
          const next = translateMessage(child.nodeValue);
          if (next !== child.nodeValue) child.nodeValue = next;
        }
      }
      document.documentElement.lang = language;
    } finally {
      applying = false;
    }
  }

  function createLanguagePicker() {
    const hosts = document.querySelectorAll('[data-i18n-language-host]');
    for (const host of hosts) {
      if (host.querySelector('select')) continue;
      const wrapper = document.createElement('label');
      wrapper.className = 'language-picker';
      const text = document.createElement('span');
      text.className = 'language-picker-label';
      text.textContent = t('Language');
      const select = document.createElement('select');
      select.setAttribute('aria-label', t('Language'));
      select.innerHTML = '<option value="en">English</option><option value="uk">Українська</option>';
      select.value = language;
      select.addEventListener('change', () => setLanguage(select.value));
      wrapper.append(text, select);
      host.append(wrapper);
    }
  }

  function refreshLanguagePickers() {
    for (const host of document.querySelectorAll('[data-i18n-language-host]')) {
      const label = host.querySelector('.language-picker-label');
      const select = host.querySelector('select');
      if (label) label.textContent = t('Language');
      if (select) {
        select.value = language;
        select.setAttribute('aria-label', t('Language'));
      }
    }
  }

  function persistNativeLanguage(code) {
    const token = document.querySelector('meta[name=\"gbavm-session-token\"]')?.content || '';
    fetch('./api/preferences/language', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-GBA-Token': token },
      body: JSON.stringify({ language: code }),
    }).catch(() => {});
  }

  function setLanguage(value) {
    language = normalizeLanguage(value);
    try { localStorage.setItem(STORAGE_KEY, language); } catch (_) {}
    persistNativeLanguage(language);
    translateNode(document);
    refreshLanguagePickers();
    window.dispatchEvent(new CustomEvent('gba-language-changed', { detail: { language } }));
  }

  async function loadCatalog(code) {
    const response = await fetch(`./locales/${code}.json`, { cache: 'no-cache' });
    if (!response.ok) throw new Error(`Could not load ${code} localization.`);
    catalogs.set(code, await response.json());
  }

  const nativeAlert = window.alert.bind(window);
  const nativeConfirm = window.confirm.bind(window);
  window.alert = (message) => nativeAlert(translateMessage(message));
  window.confirm = (message) => nativeConfirm(translateMessage(message));

  const api = {
    get language() { return language; },
    get supportedLanguages() { return [...SUPPORTED]; },
    t,
    translateMessage,
    setLanguage,
    ready: Promise.resolve(),
  };
  window.GBAMediaI18n = api;

  language = detectLanguage();
  api.ready = Promise.all(SUPPORTED.map(code => loadCatalog(code)))
    .catch(error => console.warn('Localization fallback:', error))
    .then(() => {
      createLanguagePicker();
      translateNode(document);
      const observer = new MutationObserver(records => {
        if (applying) return;
        for (const record of records) {
          if (record.type === 'characterData') translateNode(record.target);
          else for (const node of record.addedNodes) translateNode(node);
        }
      });
      observer.observe(document.body, { subtree: true, childList: true, characterData: true });
    });
})();
