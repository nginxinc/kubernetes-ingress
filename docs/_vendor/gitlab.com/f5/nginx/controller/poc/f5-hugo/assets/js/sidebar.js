// This code makes the sidebar remember which sections has been clicked when using the sidebar
$(document).ready(function () {
    $(".sidebar .nginx-toc-link a").each(function(i,item) {
        if (item.dataset.menuId == $(".main").data("menuId")) {
            $(item).css("color", "#429345");
            $(item).css("font-weight", "500");
            $(item).parents(".collapse").each(function(i,el) {
                var col = new bootstrap.Collapse(el, {
                    toggle: false
                });
                col.show();
            });
      }
      });
});