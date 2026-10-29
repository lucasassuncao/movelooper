// Sets data-theme on <html>: the reader's pick from a [data-theme-toggle]
// button, remembered across the site and its flow diagrams, or else the OS's.
(function () {
  var root = document.documentElement;
  var light = matchMedia("(prefers-color-scheme: light)");

  function saved() {
    try { return localStorage.getItem("theme"); } catch (e) { return null; }
  }
  function os() { return light.matches ? "light" : "dark"; }
  function apply() {
    var pick = saved();
    root.dataset.theme = pick === "light" || pick === "dark" ? pick : os();
  }

  apply();
  light.addEventListener("change", apply);

  document.addEventListener("click", function (e) {
    var button = e.target.closest && e.target.closest("[data-theme-toggle]");
    if (!button) return;
    var next = root.dataset.theme === "light" ? "dark" : "light";
    try {
      localStorage.setItem("theme", next);
      localStorage.setItem("archify-theme", next); // the flow diagrams' own key
    } catch (e2) { /* private mode */ }
    root.dataset.theme = next;
  });
})();
