'use strict';

(function(exports){

  var contentElem = undefined;
  var bkmID = undefined;

  function init(gmClient, _contentElem, srcDir, queryParams, curTabData) {
    contentElem = _contentElem;
    bkmID = queryParams.bkm_id * 1;
    var tagsInputElem = contentElem.find('#tags_input')

    var gmTagReqInst = gmTagRequester.create({
      tagsInputElem: tagsInputElem,
      allowNewTags: true,
      gmClient: gmClient,

      loadingStatus: function(isLoading) {
        if (isLoading) {
          contentElem.find("#tmploading").html("<p>...</p>");
        } else {
          contentElem.find("#tmploading").html("<p>&nbsp</p>");
        }
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
          ? gmClient.updateBookmark.bind(gmClient, bkmID)
          : gmClient.addBookmark.bind(gmClient);

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

          gmClient.addTag("/" + parts.join("/"), {
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

    gmClient.onConnected(true, function() {
      gmClient.getBookmarkByID(bkmID, function(status, resp) {
        console.log('getBookmarkByID resp:', status, resp);

        if (status === 200) {
          if (resp.url) {
            contentElem.find("#bkm_url").val(resp.url);
          }

          if (resp.title) {
            contentElem.find("#bkm_title").val(resp.title);
          }

          if (resp.comment) {
            contentElem.find("#bkm_comment").val(resp.comment);
          }

          resp.tags.forEach(function(tag) {
            gmTagReqInst.addTag(tag.id, tag.fullName, false)
          });
        } else {
          // TODO: show error
        }
      });
    });

    if (!bkmID) {
      contentElem.find("#bkm_url").val(curTabData.url);
      contentElem.find("#bkm_title").val(curTabData.title);
    }
  }

  exports.init = init;

})(typeof exports === 'undefined' ? this['gmEditBookmark']={} : exports);
