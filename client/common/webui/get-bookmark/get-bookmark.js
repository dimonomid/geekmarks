(function() {

  var loading = false;
  var needLoad = false;
  //var mytext = "";

  var curTagsArr = [];
  var curTagsMap = {};

  var gmClient = getBookmarkWrapper.createGMClient();

  function artificialDelay(f) {
    setTimeout(f, 150);
  }

  getBookmarkWrapper.onLoad(function() {
    //$("#tags_input").on('input', function(d) {
      //mytext = $(this).val();
      //if (!loading) {
        //send();
      //} else {
        //needLoad = true;
      //}
      //console.log($(this).val())
    //})

    $('#tags_input').tagEditor({
      autocomplete: {
        delay: 0, // show suggestions immediately
        position: { collision: 'flip' }, // automatic menu position up/down
        autoFocus: true,
        source: function(request, cb) {
          //console.log('req:', request);

          artificialDelay(function(){
            gmClient.getTagsByPattern(request.term, function(arr) {
              var i;
              curTagsArr = arr;
              curTagsMap = {};

              for (i = 0; i < arr.length; i++) {
                curTagsMap[arr[i].path] = arr[i];
              }

              //console.log('got resp to getTagsByPattern:', arr);
              cb(arr.map(
                function(item) {
                  return item.path;
                }
              ));
            });
          });
        },
        change: function(event, ui){
          console.log('change!');
          $(this).val((ui.item ? ui.item.id : ""));
        },
      },
      forceLowercase: false,
      placeholder: 'Enter tags',
      //delimiter: ' ,;',
      removeDuplicates: true,
      beforeTagSave: function(field, editor, tags, tag, val) {
        console.log("save:", val, ", curTagsArr:", curTagsArr);

        if (val in curTagsMap) {
          // Tag exists in the currently suggested tags: use it
          return val;
        } else {
          // Tag does not exist in the currently suggested tags: use
          // the first suggestion (if any)
          if (curTagsArr.length > 0) {
            return curTagsArr[0].path;
          } else {
            return false;
          }
        }
      }
    });

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
