(function () {
    var form = document.querySelector('.hc-search');
    if (!form || !window.fetch) return;

    var input = form.querySelector('input[name="q"]');
    var slug = form.getAttribute('data-hc-slug');
    var locale = form.getAttribute('data-hc-locale');
    var emptyText = form.getAttribute('data-hc-empty') || 'No results';
    var resultsText = form.getAttribute('data-hc-results') || '{count} results';
    var resultsTextOne = form.getAttribute('data-hc-results-one') || '{count} result';
    if (!input || !slug) return;

    var DEBOUNCE = 250;
    var MIN_CHARS = 2;
    var api = '/api/v1/public/help-centers/' + encodeURIComponent(slug) + '/search';
    var base = form.getAttribute('data-hc-base') || '/hc/' + slug + '/' + locale;
    var articleBase = base + '/articles/';

    var panel = document.createElement('div');
    panel.className = 'hc-typeahead';
    panel.id = 'hc-typeahead-panel';
    panel.setAttribute('role', 'listbox');
    panel.hidden = true;
    form.appendChild(panel);

    var status = document.createElement('div');
    status.className = 'hc-sr-only';
    status.setAttribute('aria-live', 'polite');
    form.appendChild(status);

    var timer = null;
    var seq = 0;
    var active = -1;
    var items = [];
    // The term shown to the reader but not yet written to the search log.
    var pending = '';

    function url(q, log) {
        return api + '?q=' + encodeURIComponent(q) + '&locale=' + encodeURIComponent(locale) + '&log=' + log;
    }

    function close() {
        seq++;
        panel.hidden = true;
        active = -1;
        input.setAttribute('aria-expanded', 'false');
        input.removeAttribute('aria-activedescendant');
    }

    // One row per search: the settled term is logged when the reader acts on it or walks away.
    function flush() {
        if (!pending) return;
        var q = pending;
        pending = '';
        // keepalive rather than sendBeacon: the log endpoint is a GET, sendBeacon only POSTs.
        fetch(url(q, '1'), { keepalive: true }).catch(function () {});
    }

    function render(list, q) {
        panel.replaceChildren();
        items = [];
        if (!list.length) {
            var none = document.createElement('p');
            none.className = 'hc-typeahead-empty';
            none.textContent = emptyText;
            panel.appendChild(none);
        }
        list.forEach(function (a, i) {
            var link = document.createElement('a');
            link.href = articleBase + a.slug;
            link.id = 'hc-typeahead-option-' + i;
            link.setAttribute('role', 'option');
            link.textContent = a.title;
            link.addEventListener('click', flush);
            panel.appendChild(link);
            items.push(link);
        });
        panel.hidden = false;
        active = -1;
        input.setAttribute('aria-expanded', 'true');
        status.textContent = list.length ? (list.length > 1 ? resultsText : resultsTextOne).replace('{count}', list.length) : emptyText;
        pending = q;
    }

    function search(q) {
        var mine = ++seq;
        fetch(url(q, '0'), { headers: { Accept: 'application/json' } })
            .then(function (res) { return res.ok ? res.json() : null; })
            .then(function (body) {
                if (mine !== seq) return;
                // A failed lookup must not leave the previous term's suggestions on screen.
                if (!body) { close(); return; }
                render(body.data || [], q);
            })
            .catch(function () { if (mine === seq) close(); });
    }

    // aria-activedescendant rather than moving focus: the caret stays in the input so typing keeps working.
    function highlight(next) {
        if (!items.length) return;
        if (active > -1) items[active].removeAttribute('aria-selected');
        active = (next + items.length) % items.length;
        items[active].setAttribute('aria-selected', 'true');
        items[active].scrollIntoView({ block: 'nearest' });
        input.setAttribute('aria-activedescendant', items[active].id);
    }

    input.setAttribute('role', 'combobox');
    input.setAttribute('aria-expanded', 'false');
    input.setAttribute('aria-autocomplete', 'list');
    input.setAttribute('aria-controls', panel.id);

    input.addEventListener('input', function () {
        var q = input.value.trim();
        clearTimeout(timer);
        seq++;
        // A term the reader abandoned mid-session still belongs in the log.
        if (pending && q.indexOf(pending) !== 0) flush();
        if (q.length < MIN_CHARS) {
            close();
            return;
        }
        timer = setTimeout(function () { search(q); }, DEBOUNCE);
    });

    form.addEventListener('keydown', function (e) {
        if (e.key === 'Escape') { close(); input.focus(); return; }
        if (panel.hidden || !items.length) return;
        if (e.key === 'ArrowDown') { e.preventDefault(); highlight(active + 1); }
        else if (e.key === 'ArrowUp') { e.preventDefault(); highlight(active - 1); }
        else if (e.key === 'Enter' && active > -1) { e.preventDefault(); items[active].click(); }
    });

    // The results page logs the submitted term itself; flushing too would double-count it.
    form.addEventListener('submit', function () {
        clearTimeout(timer);
        pending = '';
    });

    document.addEventListener('click', function (e) {
        if (!form.contains(e.target)) { flush(); close(); }
    });
    window.addEventListener('pagehide', flush);
})();
