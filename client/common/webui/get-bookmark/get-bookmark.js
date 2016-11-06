(function() {

  var gmTagRequester = (function gmTagRequester() {
    function create(opts) {

      // Default option values
      var defaults = {
        // Mandatory: input field which should be used for tags
        tagsInputElem: undefined,
        // Mandatory: gmClient instance
        gmClient: undefined,
        // If false, only the suggested tags are allowed
        allowNewTags: false,
        // Callback which will be called when request starts or finishes
        loadingStatus: function(isLoading) {},
        onChange: function(selectedTagIDs) {},
      };

      // True if tags request is in progress
      var loading = false;

      // If new request has been made before the previous one has finished,
      // pendingRequest contains the string pattern to be requested once we get
      // response to the previous request.
      var pendingRequest = undefined;

      // Current tag suggestions: array of objects, and map from path to object
      var curTagsArr = [];
      var curTagsMap = {};

      // Map from tag path to tag objects, which we've ever encountered
      var tagsByPath = {};

      // Callback which needs to be called once new tags are received
      var respCallback = undefined;

      var selectedTagIDs = [];

      var gmClient = opts.gmClient;

      opts = $.extend({}, defaults, opts);

      opts.tagsInputElem.tagEditor({
        autocomplete: {
          delay: 0, // show suggestions immediately
          position: { collision: 'flip' }, // automatic menu position up/down
          autoFocus: true,
          source: function(request, cb) {
            console.log('req:', request);

            respCallback = cb;
            if (!loading) {
              queryTags(request.term);
            } else {
              pendingRequest = request.term;
            }
          },
        },
        forceLowercase: false,
        placeholder: 'Enter tags',
        //delimiter: ' ,;',
        removeDuplicates: true,
        beforeTagSave: function(field, editor, tags, tag, val) {
          if (!opts.allowNewTags) {
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
          } else {
            return val;
          }
        },

        onChange: function(field, editor, tags) {
          // Remember IDs of selected tags
          selectedTagIDs = tags.map(function(path) {
            return tagsByPath[path].id;
          });

          // Call user's callback with selected tags
          opts.onChange(selectedTagIDs.slice());
        },
      });

      opts.tagsInputElem.focus();

      function queryTags(pattern) {
        opts.loadingStatus(true);
        pendingRequest = undefined;
        loading = true;

        console.log('requesting:', pattern);
        gmClient.getTagsByPattern(pattern, function(arr) {
          var i;

          console.log('got resp to getTagsByPattern:', arr)
          opts.loadingStatus(false);
          loading = false;

          curTagsArr = arr;
          curTagsMap = {};

          for (i = 0; i < arr.length; i++) {
            curTagsMap[arr[i].path] = arr[i];
            tagsByPath[arr[i].path] = arr[i];
          }

          respCallback(arr.map(
            function(item) {
              item = $.extend({}, item, {
                toString: function() {
                  return this.path;
                },
              });
              return {
                // TODO: implement bookmarks count
                label: item.path + " (0)",
                value: item,
              };
            }
          ));

          if (typeof(pendingRequest) === "string") {
            queryTags(pendingRequest);
          }
        });
      }

      return {};
    }

    return {
      create: create,
    };

  })();

  gmPageWrapper.onLoad(function(contentElem, srcDir) {
    var gmClient = gmPageWrapper.createGMClient();
    var tagsInputElem = contentElem.find('#tags_input')
    var gmTagReqInst = gmTagRequester.create({
      tagsInputElem: tagsInputElem,
      allowNewTags: false,
      gmClient: gmClient,

      loadingStatus: function(isLoading) {
        if (isLoading) {
          contentElem.find("#tmploading").html("<p>...</p>");
        } else {
          contentElem.find("#tmploading").html("<p>&nbsp</p>");
        }
      },

      onChange: function(selectedTagIDs) {
        gmClient.getBookmarks(selectedTagIDs, function(bookmarks) {
          console.log('resp bkm2:', bookmarks)

          var listElem = contentElem.find("#tmpdata");
          listElem.text("");

          bookmarks.forEach(function(bkm) {
            var div = jQuery('<div/>', {
              id: 'bookmark_' + bkm.id,
              class: 'bookmark-div',
            });
            div.load(
              srcDir + "/bkm.html",
              undefined,
              function() {
                div.find("#bkm_link").html(bkm.title);
                div.find("#bkm_link").attr('href', bkm.url);
                div.find("#bkm_link").attr('target', '_blank');
                div.appendTo(listElem);
                console.log('html:', div.html());
              }
            );
          });

        })
      }
    });
  })

})()
