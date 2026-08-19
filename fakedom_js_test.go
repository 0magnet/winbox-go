//go:build js && wasm

package winbox

import "syscall/js"

// A fake DOM, enough of one for this package.
//
// Node has no document, so without this the only testable code here is the
// arithmetic — and nearly everything in a window manager touches an element.
// It is built in JavaScript because the shape is a tree of plain objects and
// saying so directly is shorter than assembling it from js.FuncOf calls.
//
// Only what this package calls is implemented. Anything else is left off on
// purpose, so a test that starts depending on more of the DOM fails here
// rather than passing by accident.
const fakeDOMSource = `
(function () {
  function El(tag) {
    var e = {
      tagName: (tag || "div").toUpperCase(),
      id: "",
      title: "",
      src: "",
      textContent: "",

      className: "",
      style: {
        _props: {},
        setProperty: function (k, v) { this._props[k] = v; },
        getPropertyValue: function (k) { return this._props[k] || ""; },
        removeProperty: function (k) { delete this._props[k]; },
      },
      _classes: [],
      children: [],
      parentNode: null,
      // Listeners are recorded rather than dispatched: what the tests check is
      // that something was wired up, not that a browser would fire it.
      _listeners: [],
      clientWidth: 1280,
      clientHeight: 800,
      offsetWidth: 0,
      offsetHeight: 0,
      addEventListener: function (t) { this._listeners.push(t); },
      removeEventListener: function (t) {
        var i = this._listeners.indexOf(t);
        if (i >= 0) this._listeners.splice(i, 1);
      },
      setAttribute: function (k, v) { this[k] = v; },
      getAttribute: function (k) { return this[k]; },
      removeAttribute: function (k) { delete this[k]; },
      getBoundingClientRect: function () {
        return {left: 0, top: 0, right: 0, bottom: 0, width: 0, height: 0};
      },
      focus: function () {},
      blur: function () {},
      appendChild: function (c) {
        if (c.parentNode) {
          var k = c.parentNode.children.indexOf(c);
          if (k >= 0) c.parentNode.children.splice(k, 1);
        }
        c.parentNode = this;
        this.children.push(c);
        return c;
      },
      insertBefore: function (c, ref) {
        var i = ref ? this.children.indexOf(ref) : -1;
        if (c.parentNode) {
          var k = c.parentNode.children.indexOf(c);
          if (k >= 0) c.parentNode.children.splice(k, 1);
        }
        c.parentNode = this;
        if (i < 0) this.children.push(c); else this.children.splice(i, 0, c);
        return c;
      },
      removeChild: function (c) {
        var i = this.children.indexOf(c);
        if (i >= 0) { this.children.splice(i, 1); c.parentNode = null; }
        return c;
      },
      remove: function () { if (this.parentNode) this.parentNode.removeChild(this); },
      contains: function (n) {
        var found = false;
        (function walk(x) {
          for (var i = 0; i < x.children.length; i++) {
            if (x.children[i] === n) found = true;
            walk(x.children[i]);
          }
        })(this);
        return found || this === n;
      },
      matches: function (sel) {
        var want = sel.charAt(0) === "." ? sel.slice(1) : null;
        return want !== null ? this._classes.indexOf(want) >= 0 : this.tagName === sel.toUpperCase();
      },
      querySelector: function (sel) {
        var f = this.querySelectorAll(sel);
        return f.length ? f[0] : null;
      },
      querySelectorAll: function (sel) {
        var out = [];
        var self = this;
        (function walk(node) {
          for (var i = 0; i < node.children.length; i++) {
            var c = node.children[i];
            if (c.matches(sel)) out.push(c);
            walk(c);
          }
        })(self);
        return out;
      },
    };
    // getElementsByClassName is what getByClass uses; it returns a live-ish
    // list, which for these purposes is just an array.
    e.getElementsByClassName = function (cls) { return this.querySelectorAll("." + cls); };

    // cloneNode is how a window is built: the template element is filled once
    // and copied per window, so the copies must be independent.
    e.cloneNode = function (deep) {
      var c = El(this.tagName);
      c._classes = this._classes.slice();
      c.className = this.className;
      c.textContent = this.textContent;
      c.id = this.id;
      if (deep) {
        for (var i = 0; i < this.children.length; i++) c.appendChild(this.children[i].cloneNode(true));
      }
      return c;
    };

    // innerHTML, enough of it for the window template: a tree of <div> and
    // <span> with an unquoted class attribute and no text content. Anything
    // else throws rather than silently building the wrong tree.
    Object.defineProperty(e, "innerHTML", {
      get: function () { return this._html || ""; },
      set: function (html) {
        this._html = html;
        this.children.length = 0;
        var stack = [this];
        var re = /<(\/?)(\w+)([^>]*)>/g;
        var m;
        var consumed = 0;
        while ((m = re.exec(html)) !== null) {
          if (m.index !== consumed) {
            throw new Error("fake DOM innerHTML: text content is not supported: " + html.slice(consumed, m.index));
          }
          consumed = m.index + m[0].length;
          if (m[1] === "/") { stack.pop(); continue; }
          var el = El(m[2]);
          var cls = /class=["']?([\w-]+)/.exec(m[3]);
          if (cls) el.classList.add(cls[1]);
          stack[stack.length - 1].appendChild(el);
          stack.push(el);
        }
        if (consumed !== html.length) {
          throw new Error("fake DOM innerHTML: trailing text is not supported: " + html.slice(consumed));
        }
      },
    });
    e.classList = {
      _el: e,
      add: function () {
        for (var i = 0; i < arguments.length; i++) {
          if (this._el._classes.indexOf(arguments[i]) < 0) this._el._classes.push(arguments[i]);
        }
        this._el.className = this._el._classes.join(" ");
      },
      remove: function () {
        for (var i = 0; i < arguments.length; i++) {
          var j = this._el._classes.indexOf(arguments[i]);
          if (j >= 0) this._el._classes.splice(j, 1);
        }
        this._el.className = this._el._classes.join(" ");
      },
      contains: function (c) { return this._el._classes.indexOf(c) >= 0; },
      toggle: function (c) {
        if (this.contains(c)) { this.remove(c); return false; }
        this.add(c); return true;
      },
    };
    return e;
  }

  var doc = El("html");
  doc.documentElement = El("html");
  doc.documentElement.clientWidth = 1280;
  doc.documentElement.clientHeight = 800;
  doc.head = El("head");
  doc.body = El("body");
  doc.createElement = function (tag) { return El(tag); };
  doc.createTextNode = function (t) { var e = El("#text"); e.textContent = t; return e; };
  doc.getElementById = function (id) {
    var hit = null;
    [doc.head, doc.body].forEach(function (r) {
      (function walk(node) {
        for (var i = 0; i < node.children.length; i++) {
          if (node.children[i].id === id) hit = node.children[i];
          walk(node.children[i]);
        }
      })(r);
    });
    return hit;
  };
  doc.addEventListener = function () {};
  doc.removeEventListener = function () {};

  var win = {
    innerWidth: 1280,
    innerHeight: 800,
    devicePixelRatio: 1,
    addEventListener: function () {},
    removeEventListener: function () {},
    requestAnimationFrame: function (f) { return 0; },
    cancelAnimationFrame: function () {},
    getComputedStyle: function (el) { return {display: el.style.display || "block"}; },
  };

  globalThis.window = win;
  globalThis.document = doc;
  globalThis.__mkEl = El;
  return doc;
})()
`

// installFakeDOM installs the fake document and window and re-runs the
// package's own setup against them.
func installFakeDOM() {
	js.Global().Call("eval", fakeDOMSource)
	document = js.Global().Get("document")
	window = js.Global().Get("window")
	body = js.Global().Get("document").Get("body")
	setup()
	initRoot()
}

// rootSize is what the fake document reports as the available area, which is
// what percentage units are a percentage of.
const (
	fakeRootW = 1280
	fakeRootH = 800
)
