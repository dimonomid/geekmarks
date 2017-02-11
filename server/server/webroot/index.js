(function() {
  $(document).ready(function() {
    $(".logo-large").html("<p>" + gmLogo.getLogoDataHtml() + "</p>")

    $("#get_ext").click(function() {
      alert(""
        + "Currently there are issues with publishing the extension on the "
        + "Chrome Web Store. If you want to try it right now, you can still "
        + "install the unpacked extension: clone the Geekmarks git repository, "
        + "navigate to chrome://extensions in your Chrome, expand the developer dropdown menu "
        + "and click \"Load Unpacked Extension\", select the folder client/chrome-ext "
        + "from the repository."
      );
      return false;
    })
  })
})();
