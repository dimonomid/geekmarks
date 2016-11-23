'use strict';

(function(exports){

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
      onChange: function(selectedTags) {},
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
    var selectedNewTagPaths = [];

    var gmClient = opts.gmClient;

    opts = $.extend({}, defaults, opts);

    var editor = opts.tagsInputElem.tagEditor({
      autocomplete: {
        delay: 0, // show suggestions immediately
        position: { collision: 'flip' }, // automatic menu position up/down
        autoFocus: true,
        source: function(request, cb) {
          console.log('req:', request, 'loading:', loading);

          respCallback = cb;
          if (!loading) {
            queryTags(request.term);
          } else {
            pendingRequest = request.term;
          }
        },
      },
      maxLength: 100,
      forceLowercase: false,
      placeholder: 'Enter tags',
      //delimiter: ' ,;',
      removeDuplicates: true,
      beforeTagSave: function(field, editor, tags, tag, val) {
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
      },

      onChange: function(field, editor, tags) {
        // Remember IDs of selected tags (and paths of the tags to be created)
        selectedTagIDs = [];
        selectedNewTagPaths = [];

        tags.forEach(function(path) {
          var item = tagsByPath[path];
          if (item.id > 0) {
            // Existing tag
            selectedTagIDs.push(item.id);
          } else {
            // New tag
            selectedNewTagPaths.push(path);
          }
        })

        // Call user's callback with selected tags
        opts.onChange(getSelectedTags());

        // Apply class `tag-new` for non-existing tags
        $('li', editor).each(function(){
          var li = $(this);
          li.removeClass('tag-new');

          var path = li.find('.tag-editor-tag').html();
          if (path) {
            var item = tagsByPath[path];
            // For some reason, not only real tags are found, so
            // tagsByPath[path] might be undefined. Therefore we also check
            // that item is not undefined.
            if (item !== undefined && item.id <= 0) {
              li.addClass('tag-new');
            }
          }
        });
      },
    });

    opts.tagsInputElem.focus();

    function queryTags(pattern) {
      opts.loadingStatus(true);
      pendingRequest = undefined;
      loading = true;

      console.log('requesting:', pattern);
      gmClient.getTagsByPattern(pattern, opts.allowNewTags, function(status, arr) {
        var i;

        console.log('got resp to getTagsByPattern:', status, arr)

        opts.loadingStatus(false);
        loading = false;

        if (status === 200) {
          curTagsArr = arr;
          curTagsMap = {};

          for (i = 0; i < arr.length; i++) {
            curTagsMap[arr[i].path] = arr[i];
            tagsByPath[arr[i].path] = arr[i];
          }

          respCallback(arr.map(
            function(item) {
              var label;
              item = $.extend({}, item, {
                toString: function() {
                  return this.path;
                },
              });
              if (item.id > 0) {
                // TODO: implement bookmarks count
                label = item.path + " (0)";
              } else {
                if (typeof item.newTagsCnt === "number" && item.newTagsCnt > 1) {
                  label = item.path + " (NEW TAGS: " + item.newTagsCnt + ")";
                } else {
                  label = item.path + " (NEW TAG)";
                }
              }
              return {
                label: label,
                value: item,
              };
            }
          ));

          if (typeof(pendingRequest) === "string") {
            queryTags(pendingRequest);
          }
        } else {
          // TODO: show error
        }
      });
    }

    function addTag(id, path, blur) {
      console.log('addTag', path);

      // Before inserting the tag in the tagEditor, we should prepare the
      // environment: set curTagsMap and tagsByPath, just like they would be
      // set if user has entered the tag manually
      var tag = {
        id: id,
        path: path,
      };
      curTagsMap = {};
      curTagsMap[path] = tag;
      tagsByPath[path] = tag;

      // Actually insert the tag into the input field
      opts.tagsInputElem.tagEditor('addTag', path, blur);
    }

    function getSelectedTags() {
      return {
        tagIDs: selectedTagIDs.slice(),
        newTagPaths: selectedNewTagPaths.slice(),
      };
    }

    return {
      addTag: addTag,
      getSelectedTags: getSelectedTags,
    };
  }

  exports.create = create;

})(typeof exports === 'undefined' ? this['gmTagRequester']={} : exports);
