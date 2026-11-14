(function () {
    function highlightText(el, text, query) {
        // A one-letter word would mark that letter inside every other word.
        var words = query.trim().split(/\s+/).filter(function (word) { return word.length > 1; });
        if (!words.length) { el.textContent = text; return; }
        var pattern = new RegExp('(' + words.map(function (word) {
            return word.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        }).join('|') + ')', 'gi');
        text.split(pattern).forEach(function (part, i) {
            if (i % 2) {
                var mark = document.createElement('mark');
                mark.textContent = part;
                el.appendChild(mark);
            } else {
                el.appendChild(document.createTextNode(part));
            }
        });
    }

    var pageQuery = new URLSearchParams(window.location.search).get('q') || '';
    document.querySelectorAll('[data-hc-search-text]').forEach(function (el) {
        var text = el.textContent;
        el.replaceChildren();
        highlightText(el, text, pageQuery);
    });

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

    // A search opened as a dialog picks its first result on Enter. A page search box submits to the results page.
    var autoselect = form.hasAttribute('data-hc-autoselect');
    var timer = null;
    var seq = 0;
    var active = -1;
    var items = [];
    // The term shown to the reader but not yet written to the search log.
    var pending = '';

    function url(q, log) {
        return api + '?q=' + encodeURIComponent(q) + '&locale=' + encodeURIComponent(locale) + '&log=' + log;
    }

    function deselect() {
        if (active > -1) items[active].setAttribute('aria-selected', 'false');
        active = -1;
        input.removeAttribute('aria-activedescendant');
    }

    function close() {
        clearTimeout(timer);
        seq++;
        form.removeAttribute('data-hc-searching');
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

    function render(list, q, record) {
        input.removeAttribute('aria-activedescendant');
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
            link.tabIndex = -1;
            link.setAttribute('aria-selected', 'false');
            var title = document.createElement('span');
            title.className = 'hc-typeahead-title';
            highlightText(title, a.title, q);
            link.appendChild(title);
            if (a.collection_name) {
                var collection = document.createElement('span');
                collection.className = 'hc-typeahead-collection';
                collection.textContent = a.collection_name;
                link.appendChild(collection);
            }
            if (a.content) {
                var excerpt = document.createElement('span');
                excerpt.className = 'hc-typeahead-excerpt';
                highlightText(excerpt, a.content, q);
                link.appendChild(excerpt);
            }
            link.addEventListener('click', flush);
            panel.appendChild(link);
            items.push(link);
        });
        panel.hidden = false;
        panel.scrollTop = 0;
        active = -1;
        input.setAttribute('aria-expanded', 'true');
        if (autoselect) highlight(0);
        status.textContent = list.length ? (list.length > 1 ? resultsText : resultsTextOne).replace('{count}', list.length) : emptyText;
        if (record) pending = q;
    }

    // record is false for a term the reader did not type, so showing it again adds no log row.
    function search(q, record) {
        var mine = ++seq;
        fetch(url(q, '0'), { headers: { Accept: 'application/json' } })
            .then(function (res) { return res.ok ? res.json() : null; })
            .then(function (body) {
                if (mine !== seq) return;
                // A failed lookup must not leave the previous term's suggestions on screen.
                if (!body) { close(); return; }
                form.removeAttribute('data-hc-searching');
                render(body.data || [], q, record);
            })
            .catch(function () { if (mine === seq) close(); });
    }

    // aria-activedescendant rather than moving focus: the caret stays in the input so typing keeps working.
    function highlight(next) {
        if (!items.length) return;
        deselect();
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
        // The last results stay on screen until the new ones arrive, but Enter must not open one of them.
        deselect();
        form.setAttribute('data-hc-searching', '');
        timer = setTimeout(function () { search(q, true); }, DEBOUNCE);
    });

    form.addEventListener('keydown', function (e) {
        if (e.isComposing) return;
        if (e.key === 'Escape' && !panel.hidden) {
            // Stops the browser clearing the typed term. The event still reaches a surrounding dialog, which closes too.
            e.preventDefault();
            close();
            input.focus();
            return;
        }
        // Suggestions for the previous term stay visible while the new search runs and cannot be selected.
        if (panel.hidden || !items.length || form.hasAttribute('data-hc-searching')) return;
        if (e.key === 'ArrowDown') { e.preventDefault(); highlight(active + 1); }
        else if (e.key === 'ArrowUp') { e.preventDefault(); highlight(active < 0 ? items.length - 1 : active - 1); }
        else if (e.key === 'Enter' && active > -1) { e.preventDefault(); items[active].click(); }
    });

    // The results page logs the submitted term itself; flushing too would double-count it.
    form.addEventListener('submit', function () {
        clearTimeout(timer);
        pending = '';
    });

    document.addEventListener('click', function (e) {
        // The button that opens the search modal sits outside the form. Its click focuses the input and must not cancel that lookup.
        if (e.target.closest('[data-hcd-search-open]')) return;
        if (!form.contains(e.target)) { flush(); close(); }
    });
    // Keeps focus in the input while a suggestion is pressed. A browser that does not focus links on click would otherwise close the panel before the click lands.
    panel.addEventListener('mousedown', function (e) { e.preventDefault(); });
    form.addEventListener('focusout', function (e) {
        if (!form.contains(e.relatedTarget)) { flush(); close(); }
    });
    input.addEventListener('focus', function () {
        var q = input.value.trim();
        if (q.length >= MIN_CHARS && panel.hidden) {
            clearTimeout(timer);
            form.setAttribute('data-hc-searching', '');
            timer = setTimeout(function () { search(q, false); }, DEBOUNCE);
        }
    });
    window.addEventListener('pagehide', flush);
})();
