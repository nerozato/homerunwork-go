$(function () {
  $(
    "#monday-from, #monday-to, #tuesday-from, #tuesday-to, #wednesday-from, #wednesday-to, #thursday-from, #thursday-to, #friday-from, #friday-to, #saturday-from, #saturday-to, #sunday-from, #sunday-to"
  ).datetimepicker({
    format: "LT",
  });
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
$(".service-hours .custom-checkbox input:checkbox").on("change", function () {
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
  $(".textarea").each(function () {
    $(".textarea textarea").keyup(function () {
      var e = $(this).val().length;
      $(this).parent("div").find("span").text(e);
    });
  }),
  $(".uploaded-sort .uploaded-image .delete button").click(function (e) {
    e.preventDefault(), $(this).parent().parent().remove();
  }),
  $(".edit-service .uploaded-sort .uploaded-image").hover(
    function () {
      $(this).addClass("hover");
    },
    function () {
      $(this).removeClass("hover");
    }
  ),
  $("#show_hide_password i").on("click", function (e) {
    e.preventDefault(),
      "text" == $("#show_hide_password input").attr("type")
        ? ($("#show_hide_password input").attr("type", "password"),
          $("#show_hide_password i").addClass("fa-eye-slash"),
          $("#show_hide_password i").removeClass("fa-eye"))
        : "password" == $("#show_hide_password input").attr("type") &&
          ($("#show_hide_password input").attr("type", "text"),
          $("#show_hide_password i").removeClass("fa-eye-slash"),
          $("#show_hide_password i").addClass("fa-eye"));
  }),
  $(document).ready(function () {
    $(".client-actions input:radio").click(function () {
      $(".client-actions input:radio[name=" + $(this).attr("name") + "]")
        .parent()
        .removeClass("radio-checked"),
        $(this).parent().addClass("radio-checked");
    });
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
function getTimeZone() {
  return Intl.DateTimeFormat().resolvedOptions().timeZone;
}
function setCookie(name, value) {
  document.cookie = name + "=" + value + ";path=/;";
}
function setCookieTimeZone(cookieName) {
  setCookie(cookieName, getTimeZone());
}
function submitForm(formId, action, inputId, inputVal, doPost) {
  $("#" + inputId).val(inputVal);
  var formSelector = "#" + formId;
  $(formSelector).attr("action", action);
  if (doPost) {
    $(formSelector).attr("method", "POST");
  }
  $(formSelector).submit();
}
function submitBookingService(formId, doGet, clearDate, dateId, locationId) {
  if (doGet) {
    $(formId).attr("method", "GET");
  }
  if (clearDate) {
    $(dateId).val("");
  }
  if (locationId !== null) {
    $(locationId).val("");
  }
  $(formId).submit();
}
function submitCampaignService(formId, textId, titleId) {
  $(formId).attr("method", "GET");
  $(textId).val("");
  $(titleId).val("");
  $(formId).submit();
}
function submitFormInputs(formId, inputId, idVal, inputStep, stepVal) {
  $(inputId).val(idVal);
  $(inputStep).val(stepVal);
  $(formId).submit();
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
function setupCampaignImgWidget(containerId, description, prompt, inputName, imgUrlStr) {
  function createCampaignImgWidget(idx, divId, description, prompt, inputName, imgUrl) {
    var html = `
      <div class="upload-box row" id="upload-box-${idx}" data-id="${idx}">
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
                  <button type="button" class="btn btn-ligt icon-orange btn-trash" id="btn-trash-${idx}" data-id="${idx}" style="display: none;">
                      <i class="fas fa-trash"></i>
                  </button>
              </div>
              <input type="file" id="file-img-${idx}" name="${inputName}" class="d-none upload-box-input" data-id="${idx}" />
          </div>
      </div>
      `;
    $(divId).append(html);
  }
  $(document).on("change", ".upload-box-input", function (e) {
    const id = parseInt($(this).attr("data-id"));
    preview(this, e, id);
  });
  function preview(input, e, idx) {
    var ext = $(input).val().split(".").pop().toLowerCase();
    if ($.inArray(ext, ["gif", "png", "jpg", "jpeg"]) == -1) {
      $(input).parents(".upload-box").prepend('<div class="alert alert - danger">Please upload valid file type (jpg, jpeg, png or gif)</div>');
      setTimeout(function () {
        $(input).parents(".upload-box").find(".alert").remove();
      }, 4000);
    } else {
      if (input.files && input.files[0]) {
        var fileUrl = URL.createObjectURL(e.target.files[0]);
        $(input).parents(".upload-box").find(`#upload-box-container-${idx}`).hide();
        $(input).parents(".upload-box").find(`#upload-box-preview-${idx}`).removeClass("d-none");
        $(input).parents(".upload-box").find(`#upload-box-preview-${idx}`).attr("src", fileUrl);
        $("#btn-trash-" + idx).show();
        $("#div-upload-" + idx).hide();
      }
    }
  }
  $(document).on("click", ".btn-trash", function () {
    const id = parseInt($(this).attr("data-id"));
    $(this).parents(".upload-box").find(".upload-box-container").show();
    $(this).parents(".upload-box").find(".upload-box-preview").addClass("d-none");
    $(this).parents(".upload-box").find(".upload-box-preview").attr("src", "#");
    $(this).parents(".upload-box").find(".upload-box-input").val("");
    $("#btn-trash-" + id).hide();
  });
  if (imgUrlStr.length > 0) {
    createCampaignImgWidget(0, containerId, description, prompt, inputName, imgUrlStr);
    $("#file-img-0").parents(".upload-box").find("#upload-box-container-0").hide();
    $("#file-img-0").parents(".upload-box").find("#upload-box-preview-0").removeClass("d-none");
    $("#btn-trash-0").show();
    $("#div-upload-0").hide();
  } else {
    createCampaignImgWidget(0, containerId, description, prompt, inputName, "");
  }
}
function setupSvcImgWidget(containerId, description, prompt, inputName, imgUrlsStr, inputIdxName) {
  var servicePictureCount = 0;
  function createSvcImgWidget(idx, divId, description, prompt, inputName, imgUrl, inputIdxName, inputIdx) {
    var html = `
      <div class="upload-box row" id="upload-box-${idx}" data-id="${idx}">
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
                  <button type="button" class="btn btn-ligt icon-orange btn-trash" id="btn-trash-${idx}" data-id="${idx}" style="display: none;">
                      <i class="fas fa-trash"></i>
                  </button>
              </div>
              <input type="hidden" name="${inputIdxName}" value="${inputIdx}"/>
              <input type="file" id="file-img-${idx}" name="${inputName}" class="d-none upload-box-input" data-id="${idx}" />
          </div>
      </div>
      `;
    $(divId).append(html);
  }
  function changeOrder(currentIndex, newIndex) {
    $(`#upload-box-${currentIndex}`).attr("data-id", `${newIndex}`);
    $(`#upload-box-${currentIndex}`).attr("id", `upload-box-${newIndex}`);
    $(`#upload-box-control-${currentIndex}`).attr("for", `file-img-${newIndex}`);
    $(`#upload-box-control-${currentIndex}`).attr("id", `upload-box-control-${newIndex}`);
    $(`#upload-box-preview-${currentIndex}`).attr("id", `upload-box-preview-${newIndex}`);
    $(`#btn-trash-${currentIndex}`).attr("data-id", `${newIndex}`);
    $(`#btn-trash-${currentIndex}`).attr("id", `btn-trash-${newIndex}`);
    $(`#file-img-${currentIndex}`).attr("data-id", `${newIndex}`);
    $(`#file-img-${currentIndex}`).attr("id", `file-img-${newIndex}`);
    $(`#div-upload-${currentIndex}`).attr("id", `div-upload-${newIndex}`);
  }
  $(document).on("change", ".upload-box-input", function (e) {
    const id = parseInt($(this).attr("data-id"));
    preview(this, e, id);
  });
  function preview(input, e, idx) {
    var ext = $(input).val().split(".").pop().toLowerCase();
    if ($.inArray(ext, ["gif", "png", "jpg", "jpeg"]) == -1) {
      $(input).parents(".upload-box").prepend('<div class="alert alert - danger">Please upload valid file type (jpg, jpeg, png or gif)</div>');
      setTimeout(function () {
        $(input).parents(".upload-box").find(".alert").remove();
      }, 4000);
    } else {
      if (input.files && input.files[0]) {
        var fileUrl = URL.createObjectURL(e.target.files[0]);
        $(input).parents(".upload-box").find(`#upload-box-container-${idx}`).hide();
        $(input).parents(".upload-box").find(`#upload-box-preview-${idx}`).removeClass("d-none");
        $(input).parents(".upload-box").find(`#upload-box-preview-${idx}`).attr("src", fileUrl);
        $("#btn-trash-" + idx).show();
        $("#div-upload-" + idx).hide();
        // Check Next Upload box is existing.
        if (!$(`#upload-box-${idx + 1}`).length) {
          createSvcImgWidget(servicePictureCount, containerId, description, prompt, inputName, "", "", servicePictureCount);
          servicePictureCount++;
        }
      }
    }
  }
  $(document).on("click", ".btn-trash", function () {
    const id = parseInt($(this).attr("data-id"));
    if (servicePictureCount <= 1) {
      $(this).parents(".upload-box").find(".upload-box-container").show();
      $(this).parents(".upload-box").find(".upload-box-preview").addClass("d-none");
      $(this).parents(".upload-box").find(".upload-box-preview").attr("src", "#");
      $(this).parents(".upload-box").find(".upload-box-input").val("");
      $("#btn-trash-" + id).hide();
      servicePictureCount = 1;
    } else {
      for (var i = id + 1; i < servicePictureCount; i++) {
        changeOrder(i, i - 1);
      }
      $(this).parents(".upload-box").remove();
      servicePictureCount--;
    }
  });
  $(containerId).sortable({});
  $(containerId).disableSelection();
  if (imgUrlsStr.length > 0) {
    var imgUrls = JSON.parse(imgUrlsStr);
    for (var i = 0; i < imgUrls.length; i++) {
      createSvcImgWidget(i, containerId, description, prompt, inputName, imgUrls[i], inputIdxName, i);
      servicePictureCount++;
      $(`#file-img-${i}`).parents(".upload-box").find(`#upload-box-container-${i}`).hide();
      $(`#file-img-${i}`).parents(".upload-box").find(`#upload-box-preview-${i}`).removeClass("d-none");
      $("#btn-trash-" + i).show();
      $("#div-upload-" + i).hide();
    }
  }
  createSvcImgWidget(servicePictureCount, containerId, description, prompt, inputName, "", "", 0);
  servicePictureCount++;
}
function handleServiceType(inputId, bookingId, orderId, zoomId) {
  $(inputId).change(function () {
    var isApptOnly = $(this).val() == "on";
    toggleServiceDurations(isApptOnly);
  });
  function toggleServiceDurations(isApptOnly) {
    if (isApptOnly) {
      $(bookingId).show();
      $(bookingId).prop("disabled", false);
      $(orderId).hide();
      $(orderId).prop("disabled", true);
      $(zoomId).show();
      $(zoomId).prop("disabled", false);
    } else {
      $(bookingId).hide();
      $(bookingId).prop("disabled", true);
      $(orderId).show();
      $(orderId).prop("disabled", false);
      $(zoomId).hide();
      $(zoomId).prop("disabled", true);
    }
  }
  $(inputId).trigger("change");
}
function formatRecurrenceFreq(dateStr, freq) {
  if (freq.length == 0) {
    return "";
  } else if (freq == "One-Time Only") {
    return "";
  }
  var date = new Date(dateStr);

  //static values - must have
  var weekday = new Array(7);
  weekday[0] = "Sunday";
  weekday[1] = "Monday";
  weekday[2] = "Tuesday";
  weekday[3] = "Wednesday";
  weekday[4] = "Thursday";
  weekday[5] = "Friday";
  weekday[6] = "Saturday";
  var wom = new Array(6);
  wom[0] = "first";
  wom[1] = "second";
  wom[2] = "third";
  wom[3] = "fourth";
  wom[4] = "fifth";
  wom[5] = "sixth";
  var month = date.getMonth() + 1;
  var year = date.getFullYear();
  var dayOfMonth = date.getDate(); //day of the month
  var day = date.getDay(); //day of the week

  //get nth week of the month
  var weekOfMonth = Math.ceil((dayOfMonth - 1 - day) / 7);

  //check of the week is the last in the month
  var lastDayOfMonth = new Date(date.getFullYear(), date.getMonth() + 1, 0).getDate();
  if (lastDayOfMonth - dayOfMonth < 7) {
    wom[weekOfMonth] = "last"; //convert to the "last"
  }

  //Repeat on code
  var repeatOn = date;
  var repeatTxt = "";
  if (freq == "Weekly") {
    repeatOn.setDate(repeatOn.getDate() + 7);
    var repeatOnD = moment(repeatOn).format("dddd");
    repeatTxt = "Every " + repeatOnD;
  } else if (freq == "Every Two Weeks") {
    repeatOn.setDate(repeatOn.getDate() + 7);
    var repeatOnD = moment(repeatOn).format("dddd");
    repeatTxt = "Every Other " + repeatOnD;
  } else if (freq == "Monthly") {
    repeatTxt = "on the " + wom[weekOfMonth] + " " + weekday[date.getDay()];
  }
  if (repeatTxt.length > 0) {
    repeatTxt = "(Repeating " + repeatTxt + ")";
  }
  return repeatTxt;
}
function setupSvcLocation(selectId, inputProviderId, inputClientId, inputFlexId, locType1, locType2, zoomId) {
  function handleLocationType(locType) {
    $(zoomId).hide();
    if (locType == locType1) {
      $(inputProviderId).prop("disabled", false);
      $(inputProviderId).show();
      $(inputClientId).hide();
      $(inputFlexId).hide();
      return;
    } else if (locType == locType2) {
      $(inputProviderId).prop("disabled", true);
      $(inputProviderId).hide();
      $(inputClientId).show();
      $(inputFlexId).hide();
      return;
    } else {
      $(zoomId).show();
    }
    $(inputProviderId).prop("disabled", true);
    $(inputProviderId).hide();
    $(inputClientId).hide();
    $(inputFlexId).show();
  }
  handleLocationType($(selectId).val());
  $(selectId).change(function (event) {
    handleLocationType(this.value);
  });
}
function clipLink(linkId, copiedId) {
  $("#" + linkId).show();
  var text = document.getElementById(linkId);
  text.select();
  text.setSelectionRange(0, 99999);
  document.execCommand("copy");
  $("#" + linkId).hide();
  $(copiedId).text("copied");
}
function submitFilter(formId, inputId, inputVal) {
  $(inputId).val(inputVal);
  $(formId).submit();
}
function createSchedule(divId, schedules, errDays) {
  const MAX_PERIOD_COUNT = 3;
  $(function () {
    filterScheduleData();
    initSchedules(errDays);
    initTimePicker();
    initEvents();
  });
  function filterScheduleData() {
    schedules.forEach((s) => {
      if (s.working_hours) {
        s.working_hours.forEach((slot) => {
          var dt = moment(slot.from, ["h:mm A"]).format("HH:mm");
          slot.from = dt;
        });
      }
    });
  }
  function renderTimeSlot(schedule, item, index, day_id) {
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
      if (i === item.duration) {
        duration_options += `<option value="${i}" selected>${option}</option>`;
      } else {
        duration_options += `<option value="${i}">${option}</option>`;
      }
    }
    var fromDate = new Date(),
      time = item.from.split(/\:|\-/g);
    fromDate.setHours(time[0]);
    fromDate.setMinutes(time[1]);
    var to = "";
    if (item.from) {
      const [hours, minutes] = item.from.split(":");
      const from = parseInt(hours) * 60 + parseInt(minutes);
      to = minutesToHHMM(from + item.duration);
    }
    var disabled = "";
    if (!schedule.availability) {
      disabled = "disabled";
    }
    var action_content = "";
    if (index > 0) {
      action_content = `
          <div class="form-group">
              <a class="remove_row" data-day="${day_id}" data-time="${index}">
                  <i class="fa fa-minus-circle" aria-hidden="true"></i>
              </a>
          </div>
      `;
    }
    return `
      <div class="row align-items-center" id="time-slot-${index}">
          <div class="col-lg-2 timepick_day">
              <div class="form-group">
                  <span class="h5">${schedule.day}</span>
              </div>
          </div>
          <div class="col-6 col-lg-3 timepick_inputs timepick_inputs_from">
              <div class="form-group">
                  <div class="input-group date from-datepicker" id="timepick_from_${
                    schedule.day.toLowerCase() + "_" + index
                  }" data-target-input="nearest" data-day="${day_id}" data-time="${index}">
                      <input type="text" class="form-control datetimepicker-input" data-target="#timepick_from_${
                        schedule.day.toLowerCase() + "_" + index
                      }" value="${item.from}" required ${disabled}/>
                      <div class="input-group-append" data-target="#timepick_from_${schedule.day.toLowerCase() + "_" + index}" data-toggle="datetimepicker">
                          <div class="input-group-text"><i class="fa fa-clock-o"></i></div>
                      </div>
                  </div>
              </div>
          </div>
          <div class="col-6 col-lg-3 timepick_inputs timepick_inputs_duration">
              <div class="form-group">
                  <select class="form-control duration" data-day="${day_id}" data-time="${index}" ${disabled}>
                      ${duration_options}
                  </select>
              </div>
          </div>
          <div class="col-6 col-lg-2 timepick_inputs_to">
              <div class="form-group" id="time_to_${day_id}_${index}">
                  ${to}
              </div>
          </div>
          <div class="col-6 col-lg-2 timepick_actions text-right text-lg-center">
              ${action_content}
          </div>
      </div>
  `;
  }
  function renderScheduleForDay(s, day_id) {
    var time_content = `<div class="day-slot" id="day-slot-${day_id}">`;
    var action_content = "";
    var index = 0;
    if (s.working_hours === null) {
      s.working_hours = [];
    }
    if (s.working_hours.length === 0) {
      s.working_hours.push({
        from: "",
        duration: 60,
      });
    }
    s.working_hours.forEach((item) => {
      time_content += renderTimeSlot(s, item, index, day_id);
      index++;
    });
    time_content += `</div>`;
    return time_content;
  }
  // Init Schedules.
  function initSchedules(errDays) {
    var content = "";
    var index = 0;
    schedules.forEach((s) => {
      var buttonContent = "";
      var disabled = "";
      var checked = "";
      if (!s.availability) {
        disabled = "disabled";
        checked = "checked";
      }
      if (index <= 5) {
        buttonContent = `<div class="row">
              <div class="col-md-6"><button type="button" class="btn btn-secondary btn-block cloneRow" id="cloneRow_${index}" data-day="${index}" ${disabled}>Add Period</button></div>
              <div class="col-md-6"><button type="button" class="btn btn-secondary btn-block copyButton mt-1 mt-lg-0" id="copyButton_${index}" data-day="${index}" ${disabled}>Copy to Next Day</button></div>
          </div>`;
      } else {
        buttonContent = `<button type="button" class="btn btn-secondary btn-block cloneRow" id="cloneRow_${index}" data-day="${index}" ${disabled}>Add Period</button>`;
      }
      content += `
          <div class="day">
              <div class="time_block">
                  ${renderScheduleForDay(s, index)}
              </div>
              <div class="row actions">
                  <div class="col-12 col-lg-8 offset-lg-2">
                      ${buttonContent}                        
                      <label class="error-text mt-3" id="day_error_${index}">Please select a valid time period.</label> 
                  </div>
              </div>
              <div class="row">
                  <div class="col-12 col-lg-8 offset-lg-2">
                      <div class="form-group custom-control custom-checkbox" style="margin: 10px 0 0 0">
                          <input type="checkbox" class="custom-control-input" id="mon-unavailable-${index}" data-day="${index}" ${checked}>
                          <label class="custom-control-label" for="mon-unavailable-${index}" style="height: auto;">Unavailable</label>
                      </div>
                  </div>
              </div>
          </div>
          <hr>`;
      index++;
    });
    $(divId).append(content);
    if (errDays != null) {
      for (i = 0; i < errDays.length; i++) {
        $(`#day_error_${errDays[i]}`).addClass("active");
      }
    }
  }
  // Init Working Hours From (Time Picker.)
  function initTimePicker() {
    const pickers = $(".from-datepicker")
      .datetimepicker({
        useCurrent: false,
        format: "hh:mm a",
      })
      .on("change.datetimepicker", function (e) {
        const day_id = parseInt($(this).attr("data-day"));
        const time_id = parseInt($(this).attr("data-time"));
        if (!e.date) {
          schedules[day_id].working_hours[time_id].from = false;
          $(`#day_error_${day_id}`).addClass("active");
          return;
        }
        var time = moment(e.date).format("HH:mm");
        schedules[day_id].working_hours[time_id].from = time;
        calcScheduleTo(day_id, time_id);
      });
  }
  function initEvents() {
    // Availability Checkbox.
    $(".service-hours").on("change", ".custom-checkbox input:checkbox", function (e) {
      const day_index = $(this).attr("data-day");
      $(`#day-slot-${day_index} .timepick_inputs_from .datetimepicker-input`).prop("disabled", this.checked);
      $(`#day-slot-${day_index} .timepick_inputs_duration select`).prop("disabled", this.checked);
      schedules[day_index].availability = !this.checked;
      // Disable / Enable Buttons.
      const schedule = schedules[day_index];
      $(`#copyButton_${day_index}`).attr("disabled", this.checked);
      if (this.checked) {
        $(`#cloneRow_${day_index}`).attr("disabled", true);
      } else if (schedule.working_hours.length < MAX_PERIOD_COUNT) {
        $(`#cloneRow_${day_index}`).attr("disabled", false);
        // Check To Time is next day.
        calcScheduleTo(day_index, schedules[day_index].working_hours.length - 1);
      }
    });
    // Remove TimeSlot.
    $(".service-hours .time_block").on("click", ".remove_row", function (e) {
      const day_index = parseInt($(this).attr("data-day"));
      const time_index = parseInt($(this).attr("data-time"));
      const schedule = schedules[day_index];
      schedule.working_hours.splice(time_index, 1);
      console.log("schedule: ", schedule);
      $(`#day-slot-${day_index} #time-slot-${time_index}`).remove();
      // Checking Disable "Add Period" Button.
      if (schedule.working_hours.length < MAX_PERIOD_COUNT) {
        $(`#cloneRow_${day_index}`).attr("disabled", false);
      }
      // Check validate.
      if (isValidSchedule(schedule)) {
        $(`#day_error_${day_index}`).removeClass("active");
      } else {
        $(`#day_error_${day_index}`).addClass("active");
      }
    });
    // Copy to Next Day.
    $(".copyButton").click(function () {
      const day_index = parseInt($(this).attr("data-day"));
      const timeslots = JSON.parse(JSON.stringify(schedules[day_index].working_hours));
      schedules[day_index + 1].working_hours = timeslots;
      const nextDayIndex = day_index + 1;
      // Replace timeslot content with new hours.
      $(`#day-slot-${nextDayIndex}`).empty();
      var time_content = "";
      var index = 0;
      const s = schedules[nextDayIndex];
      s.working_hours.forEach((item) => {
        time_content += renderTimeSlot(s, item, index, nextDayIndex);
        index++;
      });
      $(`#day-slot-${nextDayIndex}`).append(time_content);
      initTimePicker();
      // Check Add Period Button.
      calcScheduleTo(nextDayIndex, s.working_hours.length - 1);
    });
    // Add New Time Slot.
    $(".cloneRow").click(function () {
      const day_index = parseInt($(this).attr("data-day"));
      // Get Last Period.
      const lastPeriod = schedules[day_index].working_hours[schedules[day_index].working_hours.length - 1];
      // Get Last Period's to time.
      var fromDate = new Date();
      var time = lastPeriod.from.split(/\:|\-/g);
      fromDate.setHours(time[0]);
      fromDate.setMinutes(time[1]);
      const [hours, minutes] = lastPeriod.from.split(":");
      const from = parseInt(hours) * 60 + parseInt(minutes);
      const toTime = from + lastPeriod.duration + 60;
      const nextFrom = minutesToHHMM(toTime);
      var dt = moment(nextFrom, ["h:mm A"]).format("HH:mm");
      const newTimeSlot = {
        from: dt,
        duration: 60,
      };
      schedules[day_index].working_hours.push(newTimeSlot);
      const itemIndex = schedules[day_index].working_hours.length - 1;
      const content = renderTimeSlot(schedules[day_index], newTimeSlot, itemIndex, day_index);
      $(`#day-slot-${day_index}`).append(content);
      // Checking Disable "Add Period" Button.
      if (itemIndex >= MAX_PERIOD_COUNT - 1) {
        $(`#cloneRow_${day_index}`).attr("disabled", true);
      }
      initTimePicker();
    });
    // Change Duration.
    $(divId).on("change", ".duration", function (e) {
      const day_index = parseInt($(this).attr("data-day"));
      const time_index = parseInt($(this).attr("data-time"));

      const schedule = schedules[day_index];
      const item = schedule.working_hours[time_index];
      item.duration = parseInt($(this).val());

      calcScheduleTo(day_index, time_index);
    });
  }
  function calcScheduleTo(day_index, time_index) {
    const schedule = schedules[day_index];
    if (isValidSchedule(schedule)) {
      $(`#day_error_${day_index}`).removeClass("active");
    } else {
      $(`#day_error_${day_index}`).addClass("active");
    }
    const item = schedule.working_hours[time_index];
    if (item.from) {
      var fromDate = new Date();
      var time = item.from.split(/\:|\-/g);
      fromDate.setHours(time[0]);
      fromDate.setMinutes(time[1]);
      const [hours, minutes] = item.from.split(":");
      const from = parseInt(hours) * 60 + parseInt(minutes);
      const toTime = from + item.duration;
      const to = minutesToHHMM(toTime);
      $(`#time_to_${day_index}_${time_index}`).html(to);
      // If Next Day, disable "Add Period" button.
      if (toTime >= 1440 || schedule.working_hours.length >= MAX_PERIOD_COUNT) {
        $(`#cloneRow_${day_index}`).attr("disabled", true);
      } else {
        $(`#cloneRow_${day_index}`).attr("disabled", false);
      }
    }
  }
  function isValidSchedule(schedule) {
    if (!schedule.availability) {
      return true;
    }
    // Check Period Times.
    var index = 0;
    for (var i = 0; i < schedule.working_hours.length; i++) {
      const item1 = schedule.working_hours[i];
      if (!item1.from) {
        return false;
      }
      const dates1 = getDatesFromTime(item1.from, item1.duration);
      for (var j = i + 1; j < schedule.working_hours.length; j++) {
        const item2 = schedule.working_hours[j];
        if (!item2.from) {
          return false;
        }
        const dates2 = getDatesFromTime(item2.from, item2.duration);
        if (dateRangeOverlaps(dates1[0], dates1[1], dates2[0], dates2[1])) {
          return false;
        }
      }
    }
    return true;
  }
  function leftPad(n) {
    return n > 9 ? "" + n : "0" + n;
  }
  function convert24To12(time) {
    const [hours, minutes] = time.split(":");
    const mins = parseInt(hours) * 60 + parseInt(minutes);
    var h = Math.floor(mins / 60);
    var m = mins % 60;
    m = m < 10 ? "0" + m : m;
    var a = "am";
    if (h >= 12) a = "pm";
    if (h > 12) h = h - 12;
    var result = h + ":" + m + " " + a;
    if (result == "0:00 am") {
      result = "12:00 am";
    }
    return result;
  }
  function minutesToHHMM(mins) {
    const total = mins % 1440;
    var h = Math.floor(total / 60);
    var m = total % 60;
    m = m < 10 ? "0" + m : m;
    var a = "am";
    if (h >= 12) a = "pm";
    if (h > 12) h = h - 12;
    var result = h + ":" + m + " " + a;
    if (result == "0:00 am") {
      result = "12:00 am";
    }
    if (mins > 1440) {
      return "<span style='font-size:11px;'>Next Day</span><br/>" + result;
    }
    return result;
  }
  function dateRangeOverlaps(a_start, a_end, b_start, b_end) {
    if (a_start <= b_start && b_start <= a_end) return true; // b starts in a
    if (a_start <= b_end && b_end <= a_end) return true; // b ends in a
    if (b_start < a_start && a_end < b_end) return true; // a in b
    return false;
  }
  function getDatesFromTime(from, duration) {
    var fromDate = new Date();
    var time = from.split(/\:|\-/g);
    fromDate.setHours(time[0]);
    fromDate.setMinutes(time[1]);
    const toDate = new Date(fromDate.getTime() + duration * 60000);
    return [fromDate, toDate];
  }
  return function () {
    var index = 0;
    const ajaxData = [];
    var isValidData = true;
    schedules.forEach((s) => {
      var isValid = isValidSchedule(s);
      if (!isValid) {
        $(`#day_error_${index}`).addClass("active");
        isValidData = false;
      } else {
        var newItem = {};
        if (s.availability) {
          var working_hours = [];
          s.working_hours.forEach((timeSlot) => {
            working_hours.push({
              from: convert24To12(timeSlot.from),
              duration: timeSlot.duration,
            });
          });
          newItem = {
            day: s.day,
            working_hours: working_hours,
            availability: s.availability,
          };
        } else {
          newItem = {
            day: s.day,
            working_hours: null,
            availability: s.availability,
          };
        }
        ajaxData.push(newItem);
      }
      index++;
    });
    if (isValidData) {
      return ajaxData;
    }
    return null;
  };
}
