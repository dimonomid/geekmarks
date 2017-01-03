'use strict';

(function(exports){

  var contentElem = undefined;
  var gmClientLoggedIn = undefined;
  var tagID = undefined;

  function init(_gmClient, _contentElem, srcDir, queryParams, curTabData) {
    contentElem = _contentElem;
    _gmClient.createGMClientLoggedIn().then(function(instance) {
      gmClientLoggedIn = instance;
      tagID = queryParams.tag_id * 1;

      contentElem.find('#myform').on('submit', function(e) {
        console.log("saving tag");
        if (tagID) {
          // Update existing tag
          gmClientLoggedIn.updateTag(tagID, {
            names: contentElem.find("#tag_names").val().split(",").map(
              function(a) { return a.trim(); }
            ),
            description: contentElem.find("#tag_description").val(),
          }, onSaveResponse)
        } else {
          // Add a new tag
          // TODO
        }

        // Prevent regular form submitting
        return false;
      });

      gmClientLoggedIn.onConnected(true, function() {
        if (tagID) {
          // We have an ID of the tag to edit
          contentElem.find("#edit_form_title").html("Edit tag");
          gmClientLoggedIn.getTag(tagID, function(status, resp) {
            console.log('getTag resp:', status, resp);

            if (status == 200) {
              applyTagData(resp);
              enableFields();
            } else {
              // TODO: show error
              alert(JSON.stringify(resp));
            }
          });
        } else {
          // There's no ID of the tag to edit: will add a new one
          contentElem.find("#edit_form_title").html("Add tag");
          enableFields();
        }
      });

      function onSaveResponse(status, resp) {
        if (status === 200) {
          console.log('saved', resp);
          window.close();
        } else {
          // TODO: show error
          alert(JSON.stringify(resp));
        }
      }

      function applyTagData(tagData) {
        contentElem.find("#tag_names").val(tagData.names.join(", "));

        if (tagData.description) {
          contentElem.find("#tag_description").val(tagData.description);
        }
      }

      function enableFields() {
        contentElem.find("#tag_names").prop('disabled', false);
        contentElem.find("#tag_description").prop('disabled', false);
        contentElem.find("#tag_submit").prop('disabled', false);
      }
    });
  }

  exports.init = init;

})(typeof exports === 'undefined' ? this['gmEditTag']={} : exports);
