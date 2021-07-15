document.addEventListener('DOMContentLoaded', function () {
  Coveo.SearchEndpoint.configureCloudV2Endpoint("", 'xxe1e9046f-585c-4518-a14a-6b986a5efffd');
  const root = document.getElementById("search");
  const searchBoxRoot = document.getElementById("searchbox");
  Coveo.initSearchbox(searchBoxRoot, "/search.html");
  var resetbtn = document.querySelector('#reset_btn');
  if (resetbtn) {
    resetbtn.onclick = function () {
      document.querySelector('.coveo-facet-header-eraser').click();
    };
  }
  Coveo.$$(root).on("querySuccess", function (e, args) {
    resetbtn.style.display = "block";
  });
  Coveo.$$(root).on('afterComponentsInitialization', function (e, data) {
    setTimeout(function () {
      document.querySelector('.CoveoOmnibox input').value = Coveo.state(root, 'q');
    }, 1000);
  });
  Coveo.init(root);
})

