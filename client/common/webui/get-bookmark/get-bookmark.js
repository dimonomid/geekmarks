(function() {

  var gmTagRequester = (function gmTagRequester() {
    function create(opts) {

      // Default option values
      var defaults = {
        // Mandatory: selector of the input field which should be used for tags
        tagsInputSel: '',
        // Mandatory: gmClient instance
        gmClient: undefined,
        // If false, only the suggested tags are allowed
        allowNewTags: false,
        // Callback which will be called when request starts or finishes
        loadingStatus: function(isLoading) {},
      };

      // True if tags request is in progress
      var loading = false;

      // If new request has been made before the previous one has finished,
      // pendingRequest contains the string pattern to be requested once we get
      // response to the previous request.
      var pendingRequest = undefined;

      // Current tag suggestions
      var curTagsArr = [];
      // Map from tag path to tag objects (stored in curTagsArr)
      var curTagsMap = {};

      // Callback which needs to be called once new tags are received
      var respCallback = undefined;

      var gmClient = opts.gmClient;

      opts = $.extend({}, defaults, opts);

      $(opts.tagsInputSel).tagEditor({
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
        }
      });

      $(opts.tagsInputSel).focus();

      function queryTags(pattern) {
        opts.loadingStatus(true);
        pendingRequest = undefined;
        loading = true;

        console.log('requesting:', pattern);
        artificialDelay(function() {
          gmClient.getTagsByPattern(pattern, function(arr) {
            var i;

            console.log('got resp to getTagsByPattern:', arr)
            opts.loadingStatus(false);
            loading = false;

            curTagsArr = arr;
            curTagsMap = {};

            for (i = 0; i < arr.length; i++) {
              curTagsMap[arr[i].path] = arr[i];
            }

            respCallback(arr.map(
              function(item) {
                return item.path;
              }
            ));

            if (typeof(pendingRequest) === "string") {
              queryTags(pendingRequest);
            }
          });
        })
      }

      return {};
    }

    function artificialDelay(f) {
      setTimeout(f, 150);
    }

    return {
      create: create,
    };

  })();

  getBookmarkWrapper.onLoad(function() {
    var gmTagReqInst = gmTagRequester.create({
      tagsInputSel: '#tags_input',
      allowNewTags: false,
      gmClient: getBookmarkWrapper.createGMClient(),
      loadingStatus: function(isLoading) {
        if (isLoading) {
          $("#tmploading").html("<p>...</p>");
        } else {
          $("#tmploading").html("<p>&nbsp</p>");
        }
      },
    });
  })

})()
