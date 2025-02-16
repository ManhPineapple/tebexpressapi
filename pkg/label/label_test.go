package label

import (
	"image/png"
	"log"
	"os"
	"tebexpressapi/pkg/label/tmp"
	"testing"
	"time"
)

// func makeLogo(t string) (image.Image, error) {
// 	dst := image.NewRGBA(image.Rect(0, 0, 100, 100))
// 	beBold, err := sfnt.Parse(bevietnamprobold.TTF)
// 	if err != nil {
// 		return nil, err
// 	}

// 	face, err := opentype.NewFace(beBold, &opentype.FaceOptions{
// 		Size:    64,
// 		DPI:     72,
// 		Hinting: font.HintingNone,
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	d := font.Drawer{
// 		Dst:  dst,
// 		Src:  UniformColor,
// 		Face: face,
// 		Dot:  fixed.P(5, 70),
// 	}
// 	d.DrawString(t)

// 	return dst, nil
// }

func TestGenerateLabel(t *testing.T) {
	dt := time.Now()

	im, err := GenerateImage(Input{
		OrderNumber: "#LX69",
		Code:        "LB00024423",
		Service:     "LB PRIORTY MAIL",
		Name:        "Van Hanh",
		Address1:    "Thaibinh",
		Address2:    "Hanoi",
		City:        "Hanoi",
		Zipcode:     "100000",
		State:       "HN",
		Country:     "US",
		Weight:      1200,
		From:        "47 Nguyen Huy Tuong\nHanoi, Viet Nam",
	}, "prod")

	t.Logf("generate label: %v", time.Since(dt))

	if err != nil {
		log.Fatal(err)
	}

	fb, err := os.Create("barcode.png")
	if err != nil {
		log.Fatal(err)
	}

	defer fb.Close()
	err = png.Encode(fb, im)
	if err != nil {
		log.Fatal(err)
	}
}

func TestDevGenerateLabel(t *testing.T) {
	dt := time.Now()

	im, err := GenerateImage(Input{
		OrderNumber: "#LX69",
		Code:        "LB0007112000437736",
		Service:     "LB PRIORTY MAIL",
		Name:        "Van Hanh Tran",
		Address1:    "Thaibinh",
		Address2:    "Hanoi",
		City:        "Hanoi",
		Zipcode:     "100000",
		State:       "HN",
		Country:     "US",
		Weight:      1200,
		From:        "47Nguyen Huy Tuong\nHanoi, Viet Nam",
	}, "development")

	t.Logf("generate label: %v", time.Since(dt))

	if err != nil {
		log.Fatal(err)
	}

	fb, err := os.Create("barcode.png")
	if err != nil {
		log.Fatal(err)
	}

	defer fb.Close()
	err = png.Encode(fb, im)
	if err != nil {
		log.Fatal(err)
	}
}

func TestGenerateLabelHasLogo(t *testing.T) {
	dt := time.Now()

	iLogo, err := makeLogo("LX")
	if err != nil {
		log.Fatal(err)
	}

	im, err := GenerateImage(Input{
		OrderNumber: "#LX69",
		LogoImage:   &iLogo,
		Code:        "LB00024423",
		Service:     "LB PRIORTY MAIL",
		Name:        "Van Hanh",
		Address1:    "Thaibinh",
		Address2:    "Hanoi",
		City:        "Hanoi",
		Zipcode:     "100000",
		State:       "HN",
		Country:     "US",
		Weight:      1200,
		From:        "Company name Company name, Address, city, Zip, State",
	}, "prod")

	t.Logf("generate label: %v", time.Since(dt))

	if err != nil {
		log.Fatal(err)
	}

	fb, err := os.Create("barcode.png")
	if err != nil {
		log.Fatal(err)
	}

	defer fb.Close()
	err = png.Encode(fb, im)
	if err != nil {
		log.Fatal(err)
	}
}

func TestLabelPreview(t *testing.T) {
	dt := time.Now()

	im, err := GenerateImagePreview(Input{
		Code:     "LB00024423",
		Service:  "LB PRIORTY MAIL",
		Name:     "Van Hanh",
		Address1: "Thaibinh",
		Address2: "Hanoi",
		City:     "Hanoi",
		Zipcode:  "100000",
		State:    "HN",
		Country:  "US",
		Weight:   1200,
		From:     "Company name Company name, Address, city, Zip, State",
	})

	t.Logf("generate label: %v", time.Since(dt))

	if err != nil {
		log.Fatal(err)
	}

	fb, err := os.Create("barcode.png")
	if err != nil {
		log.Fatal(err)
	}

	defer fb.Close()
	err = png.Encode(fb, im)
	if err != nil {
		log.Fatal(err)
	}
}

func TestLabelPreviewCustomize(t *testing.T) {
	dt := time.Now()

	log.Println(string(tmp.TIM_CUS))
	return
	iLogo, err := makeLogo("LX")
	if err != nil {
		log.Fatal(err)
	}

	im, err := GenerateImagePreview(Input{
		LogoImage: &iLogo,
		Code:      "LB00024423",
		Service:   "LB PRIORTY MAIL",
		Name:      "Van Hanh",
		Address1:  "Thaibinh",
		Address2:  "Hanoi",
		City:      "Hanoi",
		Zipcode:   "100000",
		State:     "HN",
		Country:   "US",
		Weight:    1200,
		From:      "Company name Company name, Address, city, Zip, State",
	})

	t.Logf("generate label: %v", time.Since(dt))

	if err != nil {
		log.Fatal(err)
	}

	fb, err := os.Create("barcode_1.png")
	if err != nil {
		log.Fatal(err)
	}

	defer fb.Close()
	err = png.Encode(fb, im)
	if err != nil {
		log.Fatal(err)
	}
}

func BenchmarkLabel(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		GenerateImage(Input{
			Code:     "LB00024423",
			Service:  "LB PRIORTY MAIL",
			Name:     "Van Hanh",
			Address1: "Thaibinh",
			Address2: "Hanoi",
			City:     "Hanoi",
			Zipcode:  "100000",
			State:    "HN",
			Country:  "US",
			Weight:   float64(i),
			From:     "Company name, Address, city, Zip, State",
		}, "prod")
	}
}

func BenchmarkLabelPreview(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		GenerateImagePreview(Input{
			Code:     "LB00024423",
			Service:  "LB PRIORTY MAIL",
			Name:     "Van Hanh",
			Address1: "Thaibinh",
			Address2: "Hanoi",
			City:     "Hanoi",
			Zipcode:  "100000",
			State:    "HN",
			Country:  "US",
			Weight:   float64(i),
			From:     "Company name, Address, city, Zip, State",
		})
	}
}
