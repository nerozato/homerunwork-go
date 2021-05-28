package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"path"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

//image tables
const (
	dbTableImg = "img"
)

//ImgType : image type
type ImgType int

//image types
const (
	ImgTypeLogo ImgType = iota + 1
	ImgTypeBanner
	ImgTypeSvc
	ImgTypeSvcMain
	ImgTypeTestimonial
	ImgTypeFavIcon
	ImgTypeAd
)

//ImgDimension : image dimension
type ImgDimension int

//image dimensions
const (
	ImgAdHeight          ImgDimension = 628
	ImgAdWidth           ImgDimension = 1200
	ImgAboutHeight       ImgDimension = 600
	ImgAboutWidth        ImgDimension = 800
	ImgBannerHeight      ImgDimension = 740
	ImgBannerWidth       ImgDimension = 2880
	ImgFavIconHeight     ImgDimension = 16
	ImgFavIconWidth      ImgDimension = 16
	ImgLogoHeight        ImgDimension = 300
	ImgLogoWidth         ImgDimension = 300
	ImgSvcHeight         ImgDimension = 300
	ImgSvcWidth          ImgDimension = 538
	ImgTestimonialHeight ImgDimension = 300
	ImgTestimonialWidth  ImgDimension = 300
)

//ImgJpegQuality : JPEG image quality to use when encoding
const ImgJpegQuality = 90

//Img : definition of an image
type Img struct {
	ID          *uuid.UUID `json:"-"`
	UserID      *uuid.UUID `json:"-"`
	ProviderID  *uuid.UUID `json:"-"`
	SecondaryID *uuid.UUID `json:"-"`
	Type        ImgType    `json:"-"`
	Index       int        `json:"-"`
	Path        string     `json:"-"`
	FileSrc     string     `json:"-"`
	FileResized string     `json:"-"`
	Version     int64      `json:"Version"`
}

//GetFile : get the file to use for the image
func (i *Img) GetFile() string {
	//use the resized version if available
	file := i.FileResized
	if file == "" {
		file = i.FileSrc
	}

	//add the path
	return path.Join(i.Path, file)
}

//SetFile : set the file for the image
func (i *Img) SetFile(file string) {
	//split the path from the file
	i.Path = path.Dir(file)
	i.FileSrc = path.Base(file)
}

//CreateImgSelect : create a select for loading an image
func CreateImgSelect(imgTbl string) string {
	stmt := fmt.Sprintf("BIN_TO_UUID(%[1]s.id),BIN_TO_UUID(%[1]s.user_id),BIN_TO_UUID(%[1]s.provider_id),BIN_TO_UUID(%[1]s.secondary_id),%[1]s.type,%[1]s.path,%[1]s.file_src,%[1]s.file_resized,%[1]s.idx,%[1]s.data", imgTbl)
	return stmt
}

//CreateImgJoin : create a join for loading an image
func CreateImgJoin(tbl string, tblCol string, imgTbl string, imgType ImgType, imgTypeAlt ImgType) string {
	stmt := fmt.Sprintf("LEFT JOIN %[1]s %[4]s ON %[4]s.%[3]s=%[2]s.id AND %[4]s.deleted=0", dbTableImg, tbl, tblCol, imgTbl, imgType)
	if imgTypeAlt > 0 {
		stmt = fmt.Sprintf("%[1]s AND (%[2]s.type=%[3]d OR %[2]s.type=%[4]d)", stmt, imgTbl, imgType, imgTypeAlt)
	} else {
		stmt = fmt.Sprintf("%[1]s AND %[2]s.type=%[3]d", stmt, imgTbl, imgType)
	}
	return stmt
}

//CreateImg : create an image
func CreateImg(idStr sql.NullString, userIDStr sql.NullString, providerIDStr sql.NullString, secondaryIDStr sql.NullString, imgType sql.NullInt32, filePath sql.NullString, fileSrc sql.NullString, fileResized sql.NullString, index sql.NullInt32, dataStr sql.NullString) (*Img, error) {
	//check for an id
	if !idStr.Valid {
		return nil, nil
	}

	//assume other fields are set if an id is set
	id, err := uuid.FromString(idStr.String)
	if err != nil {
		return nil, errors.Wrap(err, "parse uuid id")
	}
	userID, err := uuid.FromString(userIDStr.String)
	if err != nil {
		return nil, errors.Wrap(err, "parse uuid user id")
	}
	providerID, err := uuid.FromString(providerIDStr.String)
	if err != nil {
		return nil, errors.Wrap(err, "parse uuid provider id")
	}
	secondaryID, err := uuid.FromString(secondaryIDStr.String)
	if err != nil {
		return nil, errors.Wrap(err, "parse uuid secondary id")
	}
	var img Img
	err = json.Unmarshal([]byte(dataStr.String), &img)
	if err != nil {
		return nil, errors.Wrap(err, "unjson image")
	}
	img.ID = &id
	img.UserID = &userID
	img.ProviderID = &providerID
	img.SecondaryID = &secondaryID
	img.Type = ImgType(imgType.Int32)
	img.Index = int(index.Int32)
	img.Path = filePath.String
	img.FileSrc = fileSrc.String
	img.FileResized = fileResized.String
	return &img, nil
}

//CreateImgOrder : create the order statement to use with the image select
func CreateImgOrder(imgTbl string) string {
	stmt := fmt.Sprintf("%[1]s.type,%[1]s.idx,%[1]s.updated", imgTbl)
	return stmt
}

//SaveImg : save an image
func SaveImg(ctx context.Context, db *DB, img *Img) (context.Context, error) {
	//default the provider id if necessary
	if img.ID == nil {
		uuid, err := uuid.NewV4()
		if err != nil {
			return ctx, errors.Wrap(err, "new uuid image")
		}
		img.ID = &uuid
	}

	//json encode the image data
	imgJSON, err := json.Marshal(img)
	if err != nil {
		return ctx, errors.Wrap(err, "json image")
	}

	//save the image
	stmt := fmt.Sprintf("INSERT INTO %s(id,user_id,provider_id,secondary_id,type,idx,path,file_src,file_resized,data,processing) VALUES (UUID_TO_BIN(?),UUID_TO_BIN(?),UUID_TO_BIN(?),UUID_TO_BIN(?),?,?,?,?,?,?,0) ON DUPLICATE KEY UPDATE idx=VALUES(idx),file_src=VALUES(file_src),file_resized=VALUES(file_resized),data=VALUES(data),processing=VALUES(processing),deleted=0", dbTableImg)
	ctx, result, err := db.Exec(ctx, stmt, img.ID, img.UserID, img.ProviderID, img.SecondaryID, img.Type, img.Index, img.Path, img.FileSrc, img.FileResized, imgJSON)
	if err != nil {
		return ctx, errors.Wrap(err, "insert image")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "insert image rows affected")
	}

	//0 indicated no update, 1 an insert, 2 an update
	if count < 0 || count > 2 {
		return ctx, fmt.Errorf("unable to insert image: %s: %s: %d", img.UserID, img.SecondaryID, img.Type)
	}
	return ctx, nil
}

//DeleteImgBySecondaryIDAndType : delete an image by secondary id and type
func DeleteImgBySecondaryIDAndType(ctx context.Context, db *DB, userID *uuid.UUID, secondaryID *uuid.UUID, imgType ImgType) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET deleted=1 WHERE user_id=UUID_TO_BIN(?) AND secondary_id=UUID_TO_BIN(?) AND type=?", dbTableImg)
	ctx, _, err := db.Exec(ctx, stmt, userID, secondaryID, imgType)
	if err != nil {
		return ctx, errors.Wrap(err, "delete image")
	}
	return ctx, nil
}

//list images
func listImgs(ctx context.Context, db *DB, whereStmt string, limit int, args ...interface{}) (context.Context, []*Img, error) {
	ctx, logger := GetLogger(ctx)

	//create the final query
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(id),BIN_TO_UUID(user_id),BIN_TO_UUID(provider_id),BIN_TO_UUID(secondary_id),type,path,file_src,file_resized,idx,data FROM %s WHERE %s ORDER BY idx,created LIMIT %d", dbTableImg, whereStmt, limit)

	//list the images
	ctx, rows, err := db.Query(ctx, stmt, args...)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "select images")
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			logger.Warnw("rows close", "error", err)
		}
	}()

	//read the rows
	imgs := make([]*Img, 0, 2)
	var idStr string
	var userIDStr string
	var providerIDStr string
	var secondaryIDStr string
	var imgType ImgType
	var filePath string
	var fileSrc string
	var fileResized string
	var index int
	var dataStr string
	for rows.Next() {
		err := rows.Scan(&idStr, &userIDStr, &providerIDStr, &secondaryIDStr, &imgType, &filePath, &fileSrc, &fileResized, &index, &dataStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "rows scan images")
		}

		//parse the uuid
		id, err := uuid.FromString(idStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "parse uuid image id")
		}
		userID, err := uuid.FromString(userIDStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "parse uuid user id")
		}
		providerID, err := uuid.FromString(providerIDStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "parse uuid provider id")
		}
		secondaryID, err := uuid.FromString(secondaryIDStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "parse uuid secondary id")
		}

		//unmarshal the data
		var img Img
		err = json.Unmarshal([]byte(dataStr), &img)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "unjson image")
		}
		img.ID = &id
		img.UserID = &userID
		img.ProviderID = &providerID
		img.SecondaryID = &secondaryID
		img.Type = imgType
		img.Index = index
		img.Path = filePath
		img.FileSrc = fileSrc
		img.FileResized = fileResized
		imgs = append(imgs, &img)
	}
	return ctx, imgs, nil
}

//ListImgsByTypeAndSecondaryID : list images by secondary id and type
func ListImgsByTypeAndSecondaryID(ctx context.Context, db *DB, userID *uuid.UUID, secondaryID *uuid.UUID, imgType ImgType, limit int) (context.Context, []*Img, error) {
	whereStmt := "deleted=0 AND user_id=UUID_TO_BIN(?) AND secondary_id=UUID_TO_BIN(?) AND type=?"
	return listImgs(ctx, db, whereStmt, limit, userID, secondaryID, imgType)
}

//ListImgsToProcess : list images to process
func ListImgsToProcess(ctx context.Context, db *DB, limit int) (context.Context, []*Img, error) {
	var err error
	var imgs []*Img
	whereStmt := "deleted=0 AND processing=0 AND (file_resized IS NULL OR LENGTH(file_resized)=0)"
	ctx, err = db.ProcessTx(ctx, "list imgs process", func(ctx context.Context, db *DB) (context.Context, error) {
		//list the images
		ctx, imgs, err = listImgs(ctx, db, whereStmt, limit)
		if err != nil {
			return ctx, errors.Wrap(err, "list imgs process")
		}
		if len(imgs) == 0 {
			return ctx, nil
		}

		//mark the images as being processed
		MarkImgsProcessing(ctx, db, imgs)
		return ctx, nil
	})
	if err != nil {
		return ctx, nil, errors.Wrap(err, "list imgs process")
	}
	return ctx, imgs, nil
}

//ProcessImgSingle : process an image that is a singleton
func ProcessImgSingle(ctx context.Context, db *DB, userID *uuid.UUID, providerID *uuid.UUID, secondaryID *uuid.UUID, imgType ImgType, img *Img) (context.Context, error) {
	//delete any existing entries
	ctx, err := DeleteImgBySecondaryIDAndType(ctx, db, userID, secondaryID, imgType)
	if err != nil {
		return ctx, errors.Wrap(err, "process delete image")
	}
	if img == nil {
		return ctx, nil
	}

	//save the image
	img.UserID = userID
	img.ProviderID = providerID
	img.SecondaryID = secondaryID
	img.Type = imgType
	ctx, err = SaveImg(ctx, db, img)
	if err != nil {
		return ctx, errors.Wrap(err, "process save image")
	}
	return ctx, nil
}

//ProcessImgs : process a list of images
func ProcessImgs(ctx context.Context, db *DB, userID *uuid.UUID, providerID *uuid.UUID, secondaryID *uuid.UUID, imgType ImgType, imgs []*Img) (context.Context, error) {
	//delete any existing entries
	ctx, err := DeleteImgBySecondaryIDAndType(ctx, db, userID, secondaryID, imgType)
	if err != nil {
		return ctx, errors.Wrap(err, "process delete image")
	}

	//save the images
	for idx, img := range imgs {
		img.UserID = userID
		img.ProviderID = providerID
		img.SecondaryID = secondaryID
		img.Type = imgType
		img.Index = idx
		ctx, err = SaveImg(ctx, db, img)
		if err != nil {
			return ctx, errors.Wrap(err, "process save image")
		}
	}
	return ctx, nil
}

//MarkImgsProcessing : mark images as processing
func MarkImgsProcessing(ctx context.Context, db *DB, imgs []*Img) (context.Context, error) {
	lenImgs := len(imgs)
	if lenImgs == 0 {
		return ctx, fmt.Errorf("no images to mark")
	}

	//prepare the ids
	args := make([]interface{}, lenImgs)
	for i, img := range imgs {
		args[i] = img.ID.String()
	}

	//generate the list of parameters to use in the query
	paramsStr := fmt.Sprintf("(UUID_TO_BIN(?)%s)", strings.Repeat(",UUID_TO_BIN(?)", lenImgs-1))

	//mark the images
	stmt := fmt.Sprintf("UPDATE %s SET processing=1,processing_time=CURRENT_TIMESTAMP() WHERE processing=0 AND id in %s", dbTableImg, paramsStr)
	ctx, result, err := db.Exec(ctx, stmt, args...)
	if err != nil {
		return ctx, errors.Wrap(err, "mark images processing")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "mark images processing rows affected")
	}
	if int(count) != lenImgs {
		return ctx, fmt.Errorf("unable to mark images processing: %d: %d", count, lenImgs)
	}
	return ctx, nil
}

//GetTargetDimensions : get the target dimensions for the image type as well as if the target should be cropped
func GetTargetDimensions(imgType ImgType) (ImgDimension, ImgDimension, bool, error) {
	switch imgType {
	case ImgTypeAd:
		return ImgAdWidth, ImgAdHeight, true, nil
	case ImgTypeBanner:
		return ImgBannerWidth, ImgBannerHeight, true, nil
	case ImgTypeFavIcon:
		return ImgFavIconWidth, ImgFavIconHeight, true, nil
	case ImgTypeLogo:
		return ImgLogoWidth, ImgLogoWidth, true, nil
	case ImgTypeSvc:
		return ImgSvcWidth, ImgSvcHeight, false, nil
	case ImgTypeSvcMain:
		return ImgSvcWidth, ImgSvcHeight, true, nil
	case ImgTypeTestimonial:
		return ImgTestimonialWidth, ImgTestimonialHeight, true, nil
	}
	return 0, 0, true, fmt.Errorf("unknown image type: %d", imgType)
}

//ResizeImg : create a resized image based on the input dimensions, using the specified writer
func ResizeImg(ctx context.Context, reader io.Reader, targetWidth ImgDimension, targetHeight ImgDimension, doCrop bool, writer io.Writer) (context.Context, error) {
	ctx, logger := GetLogger(ctx)
	img, format, err := image.Decode(reader)
	if err != nil {
		return ctx, errors.Wrap(err, "image decode")
	}

	//find the dimensions
	height := img.Bounds().Max.Y - img.Bounds().Min.Y
	width := img.Bounds().Max.X - img.Bounds().Min.X
	logger.Debugw("image decode", "format", format, "width", width, "height", height, "targetWidth", targetWidth, "targetHeight", targetHeight)

	//resize the image
	var resizedImg *image.NRGBA
	if doCrop {
		resizedImg = imaging.Fill(img, int(targetWidth), int(targetHeight), imaging.Top, imaging.Lanczos)
	} else {
		resizedImg = imaging.Resize(img, 0, int(targetHeight), imaging.Lanczos)
	}

	//create a white background and draw over it
	backgroundImg := image.NewRGBA(resizedImg.Bounds())
	draw.Draw(backgroundImg, backgroundImg.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	draw.Draw(backgroundImg, backgroundImg.Bounds(), resizedImg, resizedImg.Bounds().Min, draw.Over)

	//save as a jpeg
	err = imaging.Encode(writer, backgroundImg, imaging.JPEG, imaging.JPEGQuality(ImgJpegQuality))
	if err != nil {
		return ctx, errors.Wrap(err, "image save")
	}
	return ctx, nil

}
