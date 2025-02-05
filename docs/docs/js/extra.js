// fully expand the open api schema
(() => {
  const i = setInterval(() => {
    [...document.querySelectorAll('.swagger-ui [aria-expanded="false"]')].map((el) =>
      el.click()
    ).length || clearInterval(i);
  }, 1000);
})();
