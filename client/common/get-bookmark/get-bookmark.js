(function() {

  var loading = false;
  var needLoad = false;
  var mytext = "";

  var gmClient = getBookmarkWrapper.createGMClient();

  getBookmarkWrapper.onLoad(function() {
    $("#tags_input").on('input', function(d) {
      mytext = $(this).val();
      if (!loading) {
        send();
      } else {
        needLoad = true;
      }
      console.log($(this).val())
    })

    $("#tags_input").focus();
  })

  function send() {
    $("#tmploading").html("<p>...</p>");
    loading = true;
    needLoad = false;

    gmClient.getTagsByPattern(mytext, function(arr) {
      console.log('got resp to getTagsByPattern:', arr)
      $("#tmploading").html("<p>&nbsp</p>");
      loading = false;
      //var resp = JSON.parse(resp);
      //var arr = resp.body;

      $("#tmpdata").html(
        "<p>" +
        arr.reduce(
          function(str, o){
            return str + "<li>" + o.path + "</li>"
          },
          ""
        )
        + "</p>"
      );

      if (needLoad) {
        send();
      }
    })
  }

})()
