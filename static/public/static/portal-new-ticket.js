(function () {
    var form = document.querySelector('.pt-new-ticket');
    if (!form) return;

    var subject = form.querySelector('#pt-subject');
    var message = form.querySelector('#pt-message');
    var slug = form.getAttribute('data-pt-hc-slug');
    var locale = form.getAttribute('data-pt-locale');
    var articleBase = form.getAttribute('data-pt-article-base');
    var suggestTitle = form.getAttribute('data-pt-suggest-title');
    var draftKey = 'pt-draft-' + (form.getAttribute('data-pt-article') || 'default');
    var pendingKey = draftKey + '-pending';

    var DEBOUNCE = 300;
    var MIN_CHARS = 3;
    var MAX_SUGGESTIONS = 5;

    function readDraft() {
        try {
            return JSON.parse(localStorage.getItem(draftKey) || '{}');
        } catch (e) {
            return {};
        }
    }

    function saveDraft() {
        try {
            localStorage.setItem(draftKey, JSON.stringify({
                subject: subject ? subject.value : '',
                message: message ? message.value : ''
            }));
        } catch (e) {}
    }

    function clearDraft() {
        try {
            localStorage.removeItem(draftKey);
        } catch (e) {}
    }

    try {
        if (sessionStorage.getItem(pendingKey)) {
            if (form.getAttribute('data-pt-has-error') !== 'true') clearDraft();
            sessionStorage.removeItem(pendingKey);
        }
    } catch (e) {}
    var draft = readDraft();
    if (subject && !subject.value && draft.subject) subject.value = draft.subject;
    if (message && !message.value && draft.message) message.value = draft.message;

    var saveTimer = null;
    function queueSave() {
        clearTimeout(saveTimer);
        saveTimer = setTimeout(saveDraft, 400);
    }
    if (subject) subject.addEventListener('input', queueSave);
    if (message) message.addEventListener('input', queueSave);
    form.addEventListener('submit', function () {
        clearTimeout(saveTimer);
        saveDraft();
        try { sessionStorage.setItem(pendingKey, '1'); } catch (e) {}
    });

    if (!subject || !slug || !window.fetch) return;

    var panel = document.createElement('div');
    panel.className = 'pt-suggest';
    panel.hidden = true;
    panel.setAttribute('aria-live', 'polite');
    subject.insertAdjacentElement('afterend', panel);

    var timer = null;
    var seq = 0;

    function close() {
        seq++;
        panel.hidden = true;
        panel.replaceChildren();
    }

    function render(list) {
        if (!list.length) {
            close();
            return;
        }
        panel.replaceChildren();
        var heading = document.createElement('p');
        heading.textContent = suggestTitle;
        panel.appendChild(heading);
        var ul = document.createElement('ul');
        list.slice(0, MAX_SUGGESTIONS).forEach(function (a) {
            var li = document.createElement('li');
            var link = document.createElement('a');
            link.href = articleBase + a.slug;
            link.target = '_blank';
            link.rel = 'noopener noreferrer';
            link.textContent = a.title;
            li.appendChild(link);
            ul.appendChild(li);
        });
        panel.appendChild(ul);
        panel.hidden = false;
    }

    // log=0: a ticket subject is not a help center search and must not enter the search log.
    function search(q) {
        var mine = ++seq;
        var url = '/api/v1/public/help-centers/' + encodeURIComponent(slug) + '/search?log=0&locale=' +
            encodeURIComponent(locale) + '&q=' + encodeURIComponent(q);
        fetch(url, { headers: { Accept: 'application/json' } })
            .then(function (res) { return res.ok ? res.json() : null; })
            .then(function (body) {
                if (mine !== seq) return;
                if (!body) { close(); return; }
                render(body.data || []);
            })
            .catch(function () { if (mine === seq) close(); });
    }

    subject.addEventListener('input', function () {
        var q = subject.value.trim();
        clearTimeout(timer);
        seq++;
        if (q.length < MIN_CHARS) {
            close();
            return;
        }
        timer = setTimeout(function () { search(q); }, DEBOUNCE);
    });
})();
