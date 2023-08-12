package printer

import "fmt"

const (
	banner = `
   	 __  __     ______     ______     __  __    
	/\_\_\_\   /\  == \   /\  __ \   /\ \_\ \   
	\/_/\_\/_  \ \  __<   \ \  __ \  \ \____ \  
	  /\_\/\_\  \ \_\ \_\  \ \_\ \_\  \/\_____\ 
	  \/_/\/_/   \/_/ /_/   \/_/\/_/   \/_____/ 

	`
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
	White  = "\033[97m"
)

func Banner() {
	fmt.Println(Green, banner, Reset)
}
