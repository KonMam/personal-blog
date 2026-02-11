(function() {
  var toggle = document.querySelector('.theme-toggle');
  if (!toggle) return;

  function getTheme() {
    return localStorage.getItem('theme') ||
      (window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark');
  }

  function setTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('theme', theme);
    updateIcon(theme);
  }

  function updateIcon(theme) {
    var sunIcon = toggle.querySelector('.sun-icon');
    var moonIcon = toggle.querySelector('.moon-icon');
    if (theme === 'dark') {
      sunIcon.style.display = 'block';
      moonIcon.style.display = 'none';
    } else {
      sunIcon.style.display = 'none';
      moonIcon.style.display = 'block';
    }
  }

  toggle.addEventListener('click', function() {
    var current = document.documentElement.getAttribute('data-theme');
    setTheme(current === 'dark' ? 'light' : 'dark');
  });

  // Initialize icon state
  updateIcon(getTheme());

  // Listen for OS preference changes
  window.matchMedia('(prefers-color-scheme: light)').addEventListener('change', function(e) {
    if (!localStorage.getItem('theme')) {
      setTheme(e.matches ? 'light' : 'dark');
    }
  });
})();
