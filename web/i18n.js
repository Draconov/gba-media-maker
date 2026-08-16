(() => {
  'use strict';

  const STORAGE_KEY = 'gba-media-maker-language';
  const MANIFEST_URL = './locales/index.json';
  const catalogs = new Map();
  let languages = [{ code: 'en', short: 'EN', flag: '🇬🇧', file: 'en.json' }];
  let fallbackLanguage = 'en';
  let language = 'en';
  let applying = false;
  let pickerHandlersAttached = false;

  function normalizeLanguage(value) {
    const code = String(value || '').trim().toLowerCase().split(/[-_]/)[0];
    return languages.some(item => item.code === code) ? code : fallbackLanguage;
  }

  function detectLanguage() {
    const nativeSaved = document.querySelector('meta[name="gbavm-language"]')?.content;
    if (nativeSaved) return normalizeLanguage(nativeSaved);
    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (saved) return normalizeLanguage(saved);
    } catch (_) {}
    return normalizeLanguage(navigator.language || fallbackLanguage);
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
    for (const option of languages) {
      if (option.code === fallbackLanguage) continue;
      const messages = messagesFor(option.code);
      for (const [english, translated] of Object.entries(messages)) {
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
    const translated = targetCode === fallbackLanguage ? english : translateFromEnglish(english, targetCode);
    return leading + translated + trailing;
  }

  function t(key, params = {}) {
    const english = fill(messagesFor(fallbackLanguage)[key] ?? key, params);
    return language === fallbackLanguage ? english : translateFromEnglish(english, language);
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

  function currentOption() {
    return languages.find(item => item.code === language) || languages[0];
  }

  function optionName(option) {
    return catalogs.get(option.code)?.meta?.name || option.code.toUpperCase();
  }

  function closeLanguagePicker(picker, restoreFocus = false) {
    if (!picker) return;
    const button = picker.querySelector('.language-menu-button');
    const menu = picker.querySelector('.language-menu');
    if (!button || !menu) return;
    menu.hidden = true;
    button.setAttribute('aria-expanded', 'false');
    if (restoreFocus) button.focus();
  }

  function closeAllLanguagePickers(except = null) {
    for (const picker of document.querySelectorAll('.language-picker')) {
      if (picker !== except) closeLanguagePicker(picker);
    }
  }

  function focusMenuItem(menu, direction) {
    const items = [...menu.querySelectorAll('.language-menu-option')];
    if (!items.length) return;
    const activeIndex = items.indexOf(document.activeElement);
    let nextIndex = activeIndex;
    if (direction === 'first') nextIndex = 0;
    else if (direction === 'last') nextIndex = items.length - 1;
    else if (direction === 'next') nextIndex = activeIndex < 0 ? 0 : (activeIndex + 1) % items.length;
    else if (direction === 'previous') nextIndex = activeIndex < 0 ? items.length - 1 : (activeIndex - 1 + items.length) % items.length;
    items[nextIndex].focus();
  }

  function openLanguagePicker(picker, focusSelected = false) {
    const button = picker.querySelector('.language-menu-button');
    const menu = picker.querySelector('.language-menu');
    if (!button || !menu) return;
    closeAllLanguagePickers(picker);
    menu.hidden = false;
    button.setAttribute('aria-expanded', 'true');
    if (focusSelected) {
      const selected = menu.querySelector(`[data-language="${language}"]`) || menu.querySelector('.language-menu-option');
      selected?.focus();
    }
  }

  function toggleLanguagePicker(picker) {
    const menu = picker.querySelector('.language-menu');
    if (!menu) return;
    if (menu.hidden) openLanguagePicker(picker);
    else closeLanguagePicker(picker);
  }

  function buildLanguageMenu(picker) {
    const menu = picker.querySelector('.language-menu');
    if (!menu) return;
    menu.replaceChildren();
    for (const option of languages) {
      const item = document.createElement('button');
      item.type = 'button';
      item.className = 'language-menu-option';
      item.dataset.language = option.code;
      item.setAttribute('role', 'menuitemradio');
      item.innerHTML = '<span class="language-option-code"></span><span class="language-option-name"></span><span class="language-option-flag" aria-hidden="true"></span><span class="language-option-check" aria-hidden="true">✓</span>';
      item.addEventListener('click', () => {
        setLanguage(option.code);
        closeLanguagePicker(picker, true);
      });
      menu.append(item);
    }
  }

  function updateLanguagePicker(picker) {
    const display = currentOption();
    const button = picker.querySelector('.language-menu-button');
    if (!button || !display) return;
    button.querySelector('.language-code').textContent = display.short || display.code.toUpperCase();
    button.querySelector('.language-flag').textContent = display.flag || '';
    button.dataset.language = language;
    button.setAttribute('aria-label', t('Choose language'));
    button.title = t('Choose language');

    for (const item of picker.querySelectorAll('.language-menu-option')) {
      const option = languages.find(entry => entry.code === item.dataset.language);
      if (!option) continue;
      const selected = option.code === language;
      item.querySelector('.language-option-code').textContent = option.short || option.code.toUpperCase();
      item.querySelector('.language-option-name').textContent = optionName(option);
      item.querySelector('.language-option-flag').textContent = option.flag || '';
      item.setAttribute('aria-checked', selected ? 'true' : 'false');
      item.classList.toggle('is-active', selected);
    }
  }

  function attachPickerHandlers() {
    if (pickerHandlersAttached) return;
    pickerHandlersAttached = true;
    document.addEventListener('pointerdown', event => {
      if (!event.target.closest?.('.language-picker')) closeAllLanguagePickers();
    });
    document.addEventListener('keydown', event => {
      if (event.key !== 'Escape') return;
      const openPicker = [...document.querySelectorAll('.language-picker')].find(picker => !picker.querySelector('.language-menu')?.hidden);
      if (openPicker) {
        event.preventDefault();
        closeLanguagePicker(openPicker, true);
      }
    });
  }

  function createLanguagePicker() {
    attachPickerHandlers();
    for (const host of document.querySelectorAll('[data-i18n-language-host]')) {
      if (host.querySelector('.language-picker')) continue;
      const picker = document.createElement('div');
      picker.className = 'language-picker';
      picker.innerHTML = '<button class="language-menu-button" type="button" aria-haspopup="menu" aria-expanded="false"><span class="language-code"></span><span class="language-flag" aria-hidden="true"></span><span class="language-chevron" aria-hidden="true">▾</span></button><div class="language-menu" role="menu" hidden></div>';
      const button = picker.querySelector('.language-menu-button');
      const menu = picker.querySelector('.language-menu');
      button.addEventListener('click', () => toggleLanguagePicker(picker));
      button.addEventListener('keydown', event => {
        if (event.key === 'ArrowDown') {
          event.preventDefault();
          openLanguagePicker(picker, true);
        }
      });
      menu.addEventListener('keydown', event => {
        if (event.key === 'ArrowDown') { event.preventDefault(); focusMenuItem(menu, 'next'); }
        else if (event.key === 'ArrowUp') { event.preventDefault(); focusMenuItem(menu, 'previous'); }
        else if (event.key === 'Home') { event.preventDefault(); focusMenuItem(menu, 'first'); }
        else if (event.key === 'End') { event.preventDefault(); focusMenuItem(menu, 'last'); }
      });
      buildLanguageMenu(picker);
      updateLanguagePicker(picker);
      host.append(picker);
    }
  }

  function refreshLanguagePickers() {
    for (const picker of document.querySelectorAll('[data-i18n-language-host] .language-picker')) {
      updateLanguagePicker(picker);
    }
  }

  function persistNativeLanguage(code) {
    const token = document.querySelector('meta[name="gbavm-session-token"]')?.content || '';
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

  async function loadManifest() {
    const response = await fetch(MANIFEST_URL, { cache: 'no-cache' });
    if (!response.ok) throw new Error('Could not load localization manifest.');
    const manifest = await response.json();
    const entries = Array.isArray(manifest.languages) ? manifest.languages.filter(item => item && item.code && item.file) : [];
    if (!entries.length) throw new Error('Localization manifest has no languages.');
    languages = entries.map(item => ({
      code: String(item.code).trim().toLowerCase(),
      short: String(item.short || item.code).trim().toUpperCase(),
      flag: String(item.flag || ''),
      file: String(item.file).trim(),
    }));
    fallbackLanguage = languages.some(item => item.code === manifest.fallback) ? manifest.fallback : languages[0].code;
  }

  async function loadCatalog(option) {
    const response = await fetch(`./locales/${option.file}`, { cache: 'no-cache' });
    if (!response.ok) throw new Error(`Could not load ${option.code} localization.`);
    catalogs.set(option.code, await response.json());
  }

  const nativeAlert = window.alert.bind(window);
  const nativeConfirm = window.confirm.bind(window);
  window.alert = (message) => nativeAlert(translateMessage(message));
  window.confirm = (message) => nativeConfirm(translateMessage(message));

  const api = {
    get language() { return language; },
    get supportedLanguages() { return languages.map(item => item.code); },
    t,
    translateMessage,
    setLanguage,
    ready: Promise.resolve(),
  };
  window.GBAMediaI18n = api;

  api.ready = loadManifest()
    .then(() => {
      language = detectLanguage();
      return Promise.all(languages.map(loadCatalog));
    })
    .catch(error => console.warn('Localization fallback:', error))
    .then(() => {
      language = normalizeLanguage(language);
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
