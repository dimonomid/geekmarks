'use strict';

(function(exports){

  function create(opts) {

    // Default option values
    var defaults = {
      // Mandatory: input field which should be used for tags
      tagsInputElem: undefined,
      // Mandatory: gmClientLoggedIn instance
      gmClientLoggedIn: undefined,
      // If false, only the suggested tags are allowed
      allowNewTags: false,
      // Callback which will be called when request starts or finishes
      loadingStatus: function(isLoading) {},
      onChange: function(selectedTags) {},
      onTagClick: function(e) {return true;},
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

    // Map of tag paths which are pending TODO explain what's pending
    var tagsPending = {};

    // Callback which needs to be called once new tags are received
    var respCallback = undefined;

    var selectedTagIDs = [];
    var selectedNewTagPaths = [];

    var gmClientLoggedIn = opts.gmClientLoggedIn;

    opts = $.extend({}, defaults, opts);

    var editor = opts.tagsInputElem.tagEditor({
      autocomplete: {
        delay: 0, // show suggestions immediately
        position: { collision: 'flip' }, // automatic menu position up/down
        // Setting autofocus to true breaks the logic which postpones tags
        // adding until the response witht the tag list comes.
        // TODO: set it to true, and when setting `loading` to true, remove
        // selection from the menu
        autoFocus: true,
        source: function(request, cb) {
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
        } else if (loading) {
          // Tags request is in progress: for now, add a "pending" tag,
          // which will be replaced by a real one once we get a response
          tagsPending[val] = tag;
          //return val;
          return false;
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
          if (path in tagsByPath) {
            // Tag either exists or suggested by the server as a new tag
            var item = tagsByPath[path];
            if (item.id > 0) {
              // Existing tag
              selectedTagIDs.push(item.id);
            } else {
              // New tag
              selectedNewTagPaths.push(path);
            }
          } else {
            // Tag is pending: do nothing here
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

      onTagClick: function(e) {
        var tagPath = $(this).text();
        var tag = tagsByPath[tagPath]
        if (tag) {
          return opts.onTagClick.call(this, e, tag);
        }
        return true;
      },
    });

    opts.tagsInputElem.focus();

    function queryTags(pattern) {
      opts.loadingStatus(true);
      pendingRequest = undefined;
      loading = true;

      // Before starting a new request, we need to blur existing menu
      // seleciton: this is needed for the logic which postpones tag commit
      // until the response comes.
      var input = opts.tagsInputElem.tagEditor('getInput');
      var instance = input.autocomplete("instance")
      if (instance) {
        input.autocomplete("blur")
      }

      gmClientLoggedIn.getTagsByPattern(pattern, opts.allowNewTags, function(status, arr) {
        var i;

        opts.loadingStatus(false);
        loading = false;

        if (status === 200) {
          curTagsArr = arr;
          curTagsMap = {};

          if (arr.length == 0) {
            input.tooltipster("content", "No tags match \"" + pattern + "\"");
            input.tooltipster("open");
          } else {
            input.tooltipster("close");
          }

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
                //label = item.path + " (0)";
                label = item.path;
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

          if (pattern in tagsPending) {
            //opts.tagsInputElem.tagEditor('removeTag', pattern, true);
            if (curTagsArr.length > 0) {
              opts.tagsInputElem.tagEditor('addTag', curTagsArr[0].path);
            }

            // If we should replace existing tag with the new one, do that
            // NOTE: we should call `removeTag` after `addTag`, because
            // there is an issue in tagEditor: for some reason, when
            // beforeTagSave() returns `false` for already existing tag,
            // the tag is not "committed" yet. Calling addTag makes it commit
            // the previous one as well, so removeTag can actually remove it.
            if (tagsPending[pattern] !== "") {
              opts.tagsInputElem.tagEditor('removeTag', tagsPending[pattern]);
            }
            delete tagsPending[pattern];
          }

          if (typeof(pendingRequest) === "string") {
            queryTags(pendingRequest);
          }
        } else {
          // TODO: show error
        }
      });
    }

    function addTag(id, path, blur) {
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
