// The landing page's backdrop: linked rings, like the logo's, and file and
// tree glyphs that drift up in three layers, shifted by scroll and pointer.
(function () {
  var canvas = document.getElementById("backdrop");
  if (!canvas || !canvas.getContext) return;
  var ctx = canvas.getContext("2d");
  var still = matchMedia("(prefers-reduced-motion: reduce)");
  var glyphs = ["├", "└", "─", "│", "▸", "↻", "⇄", "▤", "▢", "/", "*.jpg", "*.pdf", "~/"];
  var depths = [0.25, 0.55, 1];
  var items = [], w = 0, h = 0, dpr = 1;
  var px = 0, py = 0, tx = 0, ty = 0, last = 0;

  function colours() {
    var css = getComputedStyle(document.documentElement);
    return { accent: css.getPropertyValue("--accent").trim(), line: css.getPropertyValue("--dim").trim() };
  }
  var ink = colours();

  function seed() {
    items = [];
    var n = Math.round(Math.min(70, (w * h) / 22000));
    for (var i = 0; i < n; i++) {
      var d = depths[i % depths.length];
      var ring = Math.random() < 0.45;
      items.push({
        x: Math.random() * w,
        y: Math.random() * h,
        d: d,
        ring: ring,
        rw: (50 + Math.random() * 90) * d,
        linked: ring && Math.random() < 0.4,
        glyph: glyphs[(Math.random() * glyphs.length) | 0],
        size: 10 + 16 * d,
        speed: 4 + 10 * d,
        accent: Math.random() < 0.3
      });
    }
  }

  function resize() {
    dpr = Math.min(devicePixelRatio || 1, 2);
    w = innerWidth; h = innerHeight;
    canvas.width = w * dpr; canvas.height = h * dpr;
    canvas.style.width = w + "px"; canvas.style.height = h + "px";
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    seed();
  }

  // A stadium: a rectangle whose short sides are half circles, as in the logo.
  function stadium(x, y, rw, rh) {
    var r = rh / 2;
    ctx.beginPath();
    ctx.moveTo(x + r, y);
    ctx.lineTo(x + rw - r, y);
    ctx.arc(x + rw - r, y + r, r, -Math.PI / 2, Math.PI / 2);
    ctx.lineTo(x + r, y + rh);
    ctx.arc(x + r, y + r, r, Math.PI / 2, Math.PI * 1.5);
    ctx.closePath();
    ctx.stroke();
  }

  function draw(dt) {
    ctx.clearRect(0, 0, w, h);
    px += (tx - px) * 0.05; py += (ty - py) * 0.05;
    var scroll = scrollY;
    for (var i = 0; i < items.length; i++) {
      var it = items[i];
      it.y -= it.speed * dt;
      var y = it.y - scroll * it.d * 0.35;
      var span = h + 200;
      y = ((y % span) + span) % span - 100;
      var x = it.x + px * it.d * 18;
      y += py * it.d * 12;
      ctx.globalAlpha = 0.08 + 0.14 * it.d;
      ctx.strokeStyle = ctx.fillStyle = it.accent ? ink.accent : ink.line;
      if (it.ring) {
        var rh = it.rw * 0.56;
        ctx.lineWidth = 1 + 2 * it.d;
        stadium(x, y, it.rw, rh);
        if (it.linked) {
          ctx.strokeStyle = it.accent ? ink.line : ink.accent;
          stadium(x + it.rw * 0.4, y + rh * 0.5, it.rw, rh);
        }
      } else {
        ctx.font = it.size + "px 'JetBrains Mono', monospace";
        ctx.fillText(it.glyph, x, y);
      }
    }
    ctx.globalAlpha = 1;
  }

  function frame(now) {
    var dt = last ? Math.min((now - last) / 1000, 0.05) : 0;
    last = now;
    draw(dt);
    if (!still.matches && !document.hidden) requestAnimationFrame(frame);
  }

  function start() {
    last = 0;
    if (still.matches) draw(0); else requestAnimationFrame(frame);
  }

  addEventListener("resize", function () { resize(); if (still.matches) draw(0); });
  addEventListener("pointermove", function (e) {
    tx = e.clientX / w - 0.5; ty = e.clientY / h - 0.5;
  });
  addEventListener("scroll", function () { if (still.matches) draw(0); }, { passive: true });
  document.addEventListener("visibilitychange", function () { if (!document.hidden) start(); });
  still.addEventListener("change", start);
  new MutationObserver(function () { ink = colours(); if (still.matches) draw(0); })
    .observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme"] });

  resize();
  start();
})();
