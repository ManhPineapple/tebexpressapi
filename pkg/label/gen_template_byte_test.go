package label

import (
	"testing"
)

func TestGenTemplateByte(t *testing.T) {
	ImageToByteGo("image_template/label.jpg", "tmp/base.go", "TIM")
	ImageToByteGo("image_template/label_customize.jpg", "tmp/customer.go", "TIM_CUS")
	ImageToByteGo("image_template/label_preview.jpg", "tmp/preview.go", "TIM_PRE")
	ImageToByteGo("image_template/label_customize_preview.jpg", "tmp/preview_custom.go", "TIM_CUS_PRE")
}
