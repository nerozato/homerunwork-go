$(document).on("click", "#ueberTab a", function (t) {
  for (otherTabs = $(this).attr("data-secondary").split(","), i = 0; i < otherTabs.length; i++)
    (nav = $('<ul class="nav d-none" id="tmpNav"></ul>')),
      nav.append('<li class="nav-item"><a href="#" data-toggle="tab" data-target="' + otherTabs[i] + '">nav</a></li>"'),
      nav.find("a").tab("show");
});
$("body").on("click", '[data-toggle="callout"]', function () {
  var id = $(this).attr("data-id");
  $("#" + id).toggle();
});
$(document).click(function (e) {
  var container = $('[data-toggle="callout"], .alert-callout , .alert-callout *');
  if (!container.is(e.target) && container.has(e.target).length === 0) {
    $(".alert-callout").hide();
  }
});
$(".btnNext").click(function () {
  $(".nav-tabs .active").parent().next("li").find("a").trigger("click");
}),
  $(".btnPrevious").click(function () {
    $(".nav-tabs .active").parent().prev("li").find("a").trigger("click");
  }),
  $(function () {
    $(
      "#monday-from, #monday-to, #tuesday-from, #tuesday-to, #wednesday-from, #wednesday-to, #thursday-from, #thursday-to, #friday-from, #friday-to, #saturday-from, #saturday-to, #sunday-from, #sunday-to"
    ).datetimepicker({
      format: "LT",
    });
  }),
  $(".step3 .custom-checkbox input:checkbox").on("change", function () {
    $(this).parent().parent().parent().parent().find("input:text").prop("disabled", this.checked);
    var from = "";
    var to = "";
    if (!this.checked) {
      from = "9:00 AM";
      to = "5:00 PM";
    }
    $(this).parent().parent().parent().parent().find("input:text[data-target*='from']").val(from);
    $(this).parent().parent().parent().parent().find("input:text[data-target*='to']").val(to);
  }),
  $(function () {
    $(".textarea").each(function () {
      $(".textarea textarea").keyup(function () {
        var t = $(this).val().length;
        $(this).parent("div").find("span").text(t);
      });
    });
  }),
  $("#show_hide_password i").on("click", function (t) {
    t.preventDefault(),
      "text" == $("#show_hide_password input").attr("type")
        ? ($("#show_hide_password input").attr("type", "password"),
          $("#show_hide_password i").addClass("fa-eye-slash"),
          $("#show_hide_password i").removeClass("fa-eye"))
        : "password" == $("#show_hide_password input").attr("type") &&
          ($("#show_hide_password input").attr("type", "text"),
          $("#show_hide_password i").removeClass("fa-eye-slash"),
          $("#show_hide_password i").addClass("fa-eye"));
  });
$(function () {
  $('[data-toggle="popover"]').popover({
    trigger: "focus",
  });
});
$("#free-service").change(function () {
  var value = $(this).prop("checked");
  if (value == true) {
    $("#service-price").val("0");
    $("#service-price").attr("readonly", "true");
  } else {
    $("#service-price").removeAttr("readonly", "false");
  }
});
$("#free-service-add").change(function () {
  var value = $(this).prop("checked");
  if (value == true) {
    $("#service-price-add").val("0");
    $("#service-price-add").attr("readonly", "true");
  } else {
    $("#service-price-add").removeAttr("readonly", "false");
  }
});
function initTimePicker() {
  const picker = $(".from-datepicker").datetimepicker({
    useCurrent: false,
    format: "hh:mm a",
  });
}
function renderDuration(selectedVal) {
  // Duration.
  var duration_options = "";
  for (var i = 60; i <= 720; i += 30) {
    const hour = parseInt(i / 60);
    const minute = i % 60;
    var option = "";
    if (hour === 1) {
      option = `${hour} hr `;
    } else {
      option = `${hour} hrs `;
    }
    if (minute !== 0) {
      option += `${minute} mins`;
    }
    if (i === selectedVal) {
      duration_options += `<option value="${i}" selected>${option}</option>`;
    } else {
      duration_options += `<option value="${i}">${option}</option>`;
    }
  }
  $(".duration").append(duration_options);
}
function getTimeZone() {
  return Intl.DateTimeFormat().resolvedOptions().timeZone;
}
function setCookie(name, value) {
  document.cookie = name + "=" + value + ";path=/;";
}
function setCookieTimeZone(cookieName) {
  setCookie(cookieName, getTimeZone());
}
function submitOauth(formId, inputOauthId, inputOauthIdVal, inputOauthToken, inputOauthTokenVal) {
  $("#" + inputOauthId).val(inputOauthIdVal);
  $("#" + inputOauthToken).val(inputOauthTokenVal);
  $("#" + formId).submit();
}
function setupServiceDurations(inputId, bookingClass, orderClass) {
  $(inputId).change(function () {
    var isApptOnly = $(this).val() == "on";
    toggleServiceDurations(isApptOnly);
  });
  function toggleServiceDurations(isApptOnly) {
    if (isApptOnly) {
      $(bookingClass).show();
      $(bookingClass).prop("disabled", false);
      $(orderClass).hide();
      $(orderClass).prop("disabled", true);
    } else {
      $(bookingClass).hide();
      $(bookingClass).prop("disabled", true);
      $(orderClass).show();
      $(orderClass).prop("disabled", false);
    }
  }
  $(inputId).trigger("change");
}
var cropper;
function createImgWidget(idx, divId, description, prompt, inputName, imgUrl) {
  var html = `
  <div class="upload-box row">
      <div class="col-12">
          <p>${description}</p>
          <label for="file-img-${idx}" class="upload-box-control" id="upload-box-control-${idx}">
              <div id="div-upload-${idx}" class="upload-box-container">
                  <i class="fas fa-upload"></i>
                  ${prompt}
              </div>
              <img class="upload-box-preview" id="upload-box-preview-${idx}" src="${imgUrl}" />
          </label>
          <div class="delete">
              <button type="button" class="btn btn-ligt icon-orange" id="btn-trash-${idx}"><i class="fas fa-trash"></i></button>
          </div>
          <input type="hidden" id="file-input-${idx}" name="${inputName}" />
          <input type="file" id="file-img-${idx}" class="d-none upload-box-input" />
      </div>
  </div>
  `;
  $(divId).html(html);
  $("#file-img-" + idx).on("change", function (e) {
    preview(this, e);
  });
  $("#btn-trash-" + idx).on("click", function () {
    $(this).parents(".upload-box").find(".upload-box-container").show();
    $(this).parents(".upload-box").find(".upload-box-preview").addClass("d-none");
    $(this).parents(".upload-box").find(".upload-box-preview").attr("src", "#");
    $(this).parents(".upload-box").find(".upload-box-input").val("");
    $("#btn-trash-" + idx).hide();
  });
  function preview(input, e) {
    var ext = $(input).val().split(".").pop().toLowerCase();
    if ($.inArray(ext, ["gif", "png", "jpg", "jpeg"]) == -1) {
      $(input).parents(".upload-box").prepend('<div class="alert alert - danger">Please upload valid file type (jpg, jpeg, png or gif)</div>');
      setTimeout(function () {
        $(input).parents(".upload-box").find(".alert").remove();
      }, 4000);
    } else if (input.files && input.files[0]) {
      var fileUrl = URL.createObjectURL(e.target.files[0]);
      if (cropper) {
        cropper.destroy();
        cropper = null;
      }
      $("#crop-image").removeAttr("src");
      $("#crop-image")
        .on("load", function () {
          $("#cropper-modal").modal("show");
        })
        .on("error", function () {
          alert("error loading image");
        })
        .attr("data-index", idx)
        .attr("src", fileUrl);
    }
  }
  if (imgUrl.length > 0) {
    $(`#upload-box-container-${idx}`).hide();
    $(`#div-upload-${idx}`).hide();
  }
}
function enableImgCropper(w1, h1, w2, h2) {
  $(document).on("shown.bs.modal", "#cropper-modal", function () {
    const idx = parseInt($("#crop-image").attr("data-index"));
    var ratio = parseInt(w1) / parseInt(h1);
    if (idx === 1) {
      ratio = parseInt(w2) / parseInt(h2);
    }
    const cropImage = document.getElementById("crop-image");
    cropper = new Cropper(cropImage, {
      viewMode: 1,
      aspectRatio: ratio,
      ready() {
        var width = cropImage.naturalWidth;
        var height = width;
        if (idx === 1) {
          width = cropImage.naturalWidth;
          height = width / ratio;
        }
        cropper.setCropBoxData({
          width: width,
          height: height,
        });
      },
    });
  });
  $(".btn-crop").click(function (e) {
    const idx = $("#crop-image").attr("data-index");
    var width = parseInt(w1);
    var height = parseInt(h1);
    if (parseInt(idx) === 1) {
      width = parseInt(w2);
      height = parseInt(h2);
    }
    const dataURL = cropper
      .getCroppedCanvas({
        width: width,
        height: height,
        fillColor: "#fff",
      })
      .toDataURL("image/jpeg", 0.8);
    $(`#upload-box-container-${idx}`).hide();
    $(`#upload-box-preview-${idx}`).removeClass("d-none");
    $(`#upload-box-preview-${idx}`).attr("src", dataURL);
    $(`#btn-trash-${idx}`).show();
    $(`#div-upload-${idx}`).hide();
    $(`#btn-trash-${idx}`).show();
    $(`#file-input-${idx}`).val(dataURL);
  });
}
