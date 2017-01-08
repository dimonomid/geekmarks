'use strict';

(function(exports){

  var contentElem = undefined;
  var bkmID = undefined;
  var gmClientLoggedIn = undefined;
  var curTabData = undefined;

  function init(_gmClient, _contentElem, srcDir, queryParams, _curTabData) {
    contentElem = _contentElem;
    curTabData = _curTabData;
    bkmID = queryParams.bkm_id * 1;

    _gmClient.createGMClientLoggedIn().then(function(instance) {
      if (instance) {
        initLoggedIn(instance, contentElem, srcDir);
      } else {
        gmPageWrapper.openPageLogin("openPageEditBookmarks", [bkmID]);
        gmPageWrapper.closeCurrentWindow();
      }
    });
  }

  function initLoggedIn(instance, contentElem, srcDir) {
    gmClientLoggedIn = instance;
    var tagsInputElem = contentElem.find('#tags_input')

    var gmTagReqInst = gmTagRequester.create({
      tagsInputElem: tagsInputElem,
      allowNewTags: true,
      gmClientLoggedIn: gmClientLoggedIn,

      loadingStatus: function(isLoading) {
        //if (isLoading) {
        //contentElem.find("#tmploading").html("<p>...</p>");
        //} else {
        //contentElem.find("#tmploading").html("<p>&nbsp</p>");
        //}
      },
    });

    contentElem.find('#myform').on('submit', function(e) {
      //alert(gmTagReqInst.getSelectedTags);
      var selTags = gmTagReqInst.getSelectedTags();
      //alert(JSON.stringify(selTags));
      //return false;

      var tagIDs = selTags.tagIDs;

      var saveBookmark = function() {
        console.log("saving bookmark");
        var saveFunc = bkmID
          ? gmClientLoggedIn.updateBookmark.bind(gmClientLoggedIn, bkmID)
          : gmClientLoggedIn.addBookmark.bind(gmClientLoggedIn);

        saveFunc({
          url: contentElem.find("#bkm_url").val(),
          title: contentElem.find("#bkm_title").val(),
          comment: contentElem.find("#bkm_comment").val(),
          tagIDs: tagIDs,
        }, function(status, resp) {
          if (status === 200) {
            console.log('saved', resp);
            window.close();
          } else {
            // TODO: show error
            alert(JSON.stringify(resp));
          }
        });
      };

      if (selTags.newTagPaths.length == 0) {
        // No new tags
        saveBookmark();
      } else {
        // There are some new tags
        var addedCnt = 0;
        selTags.newTagPaths.forEach(function(curPath) {
          console.log("adding new tag", curPath, "...");
          // TODO: use PUT request, when it's implemented, and avoid this
          // hackery with the last item

          var parts = curPath.split("/");
          // Remove the first (empty) item
          parts.splice(0, 1);

          // Remove the last item (to be given differently to POST request)
          var names = parts.splice(parts.length - 1);

          gmClientLoggedIn.addTag(parts.join("/"), {
            names: names,
            createIntermediary: true,
          }, function(status, resp) {
            console.log("tag", curPath, "adding result:", status, resp);
            if (status === 200) {
              addedCnt++;
              tagIDs.push(resp.tagID);
              console.log("current tagIDs:", tagIDs);
              if (addedCnt == selTags.newTagPaths.length) {
                saveBookmark();
              }
            } else {
              //TODO: show error
              alert(JSON.stringify(resp));
            }
          });
        });
      }

      // Prevent regular form submitting
      return false;
    });

    gmClientLoggedIn.onConnected(true, function() {
      if (bkmID) {
        contentElem.find("#edit_form_title").html("Edit bookmark");
        // We have an ID of the bookmark to edit
        gmClientLoggedIn.getBookmarkByID(bkmID, function(status, resp) {
          console.log('getBookmarkByID resp:', status, resp);

          if (status == 200) {
            applyBookmarkData(resp);
            enableFields();
          } else {
            // TODO: show error
            alert(JSON.stringify(resp));
          }
        });
      } else {
        // There's no ID of the bookmark to edit: let's check if there is a
        // bookmark with the URL of the current tab
        gmClientLoggedIn.getBookmarksByURL(curTabData.url, function(status, resp) {
          console.log('getBookmarksByURL resp:', status, resp);

          if (status === 200) {
            if (resp.length == 0 || curTabData.url === '') {
              // No existing bookmarks with the given URL: set data from
              // the current tab
              contentElem.find("#bkm_url").val(curTabData.url);
              contentElem.find("#bkm_title").val(curTabData.title);
              contentElem.find("#edit_form_title").html("Create bookmark");

            } else {
              bkmID = resp[0].id;
              if (resp.length > 1) {
                // TODO: show error
                alert('There are more than 1 bookmark with the given URL. Something is wrong :(');
              }
              applyBookmarkData(resp[0]);
              contentElem.find("#edit_form_title").html("Edit bookmark");
            }
            enableFields();
          } else {
            // TODO: show error
            alert(JSON.stringify(resp));
          }
        });
      }
    });

    function applyBookmarkData(bkmData) {
      if (bkmData.url) {
        contentElem.find("#bkm_url").val(bkmData.url);
      }

      if (bkmData.title) {
        contentElem.find("#bkm_title").val(bkmData.title);
      }

      if (bkmData.comment) {
        contentElem.find("#bkm_comment").val(bkmData.comment);
      }

      if (bkmData.tags !== undefined) {
        bkmData.tags.forEach(function(tag) {
          gmTagReqInst.addTag(tag.id, tag.fullName, false)
        });
      }
    }

    function enableFields() {
      contentElem.find("#bkm_url").prop('disabled', false);
      contentElem.find("#bkm_title").prop('disabled', false);
      contentElem.find("#bkm_comment").prop('disabled', false);
      contentElem.find("#bkm_submit").prop('disabled', false);
    }
  }

  exports.init = init;

})(typeof exports === 'undefined' ? this['gmEditBookmark']={} : exports);
