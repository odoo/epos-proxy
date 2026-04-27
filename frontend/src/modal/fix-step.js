import image1 from "../assets/zadig/zadig1.jpg"
import image2 from "../assets/zadig/zadig2.jpg"
import image3 from "../assets/zadig/zadig3.jpg"
import cups_admin from "../assets/cups/cups_admin.png";
import select_printer from "../assets/cups/select_printer.png";
import select_driver from "../assets/cups/select_driver.png";
import name_desc_loc from "../assets/cups/name_desc_loc.png";

const CUPS_STEPS = [
    {
        title: "Open CUPS Admin Panel",
        desc: "Open your browser and go to *http://localhost:631*. This is the CUPS web interface.",
        link: "http://localhost:631",
        linkLabel: "Open CUPS Admin Panel",
    },
    {
        title: "Go to Administration",
        desc: "Click on the *Administration* tab in the CUPS panel. You may be asked to enter your system *username* and *password*.",
        image: cups_admin,
    },
    {
        title: "Add Printer",
        desc: "Click on *Add Printer*",
    },
    {
        title: "Select Printer",
        desc: "Choose your printer from the *Local Printer* list and click *Continue*.",
        image: select_printer,
    },
    {
        title: "Configure Printer",
        desc: "Set printer *Name*, *Description*, and *Location*. then click *Continue*.",
        image: name_desc_loc,
    },
    {
        title: "Choose Driver",
        desc: "Select the appropriate *driver* from the list *OR* provide a *PPD file* if required.",
        image: select_driver,
    },
    {
        title: "Finish Setup",
        desc: "Click *Add Printer*",
    },
];

const windowsPrinterSteps = [
    {
        title: "Open Printer Settings",
        desc: "Press *Win + I* to open Settings, then go to *Bluetooth & devices* → *Printers & scanners*.",
    },
    {
        title: "Add a Printer",
        desc: "Click on *Add device* next to *Add a printer or scanner*.",
    },
    {
        title: "Select Your Printer",
        desc: "Wait for Windows to scan and select your printer from the list.\nIf not visible, click *Add manually*.",
    },
    {
        title: "Manual Setup (if needed)",
        desc: "Choose *Add a local printer or network printer with manual settings*.\nSelect the correct port (USB / IP) and continue.",
    },
    {
        title: "Install Driver",
        desc: "Select the appropriate driver from the list or install using the manufacturer’s driver.",
    },
    {
        title: "Finish Setup",
        desc: "Complete the setup and ensure the printer appears in *Printers & scanners*.\nYour app will now detect it via WMI.",
    },
];

export function windowsSteps() {
    return {
        THERMAL: [
            {
                title: "Download Zadig",
                desc: "Download the latest version of Zadig from the official website.",
                link: "https://zadig.akeo.ie",
                linkLabel: "Open Zadig website",
            },
            {
                title: "Open Zadig",
                desc: "Run *Zadig.exe* as *Administrator*.\nGo to Options → *List All Devices.*",
                image: image1,
            },
            {
                title: "Select your printer",
                desc: `Find the printer in the dropdown list.\n The name may vary, but it should look similar to your printer name.`,
                image: image2,
            },
            {
                title: "Install WinUSB driver",
                desc: "Make sure *WinUSB* is selected, then click *Replace Driver*. Wait for completion.",
                image: image3,
            },
        ],
        OFFICE: windowsPrinterSteps,
    };
}

export function macSteps() {
    return {
        OFFICE: CUPS_STEPS,
        THERMAL: [
            {
                title: "Install Homebrew",
                desc: "Homebrew is a package manager for macOS. If you already have it, skip to the next step.",
                link: "https://brew.sh",
                linkLabel: "Open Homebrew website",
            },
            {
                title: "Install libusb",
                desc: "Once Homebrew is installed, run this command in Terminal:",
                codes: ["brew install libusb"],
            },
            {
                title: "Reconnect your printer",
                desc: `Unplug and replug your printer name.\nThe driver should now be detected automatically.`,
            },
        ],
    };
}

export function linuxSteps() {
    return {
        THERMAL: [
            {
                title: "Install libusb",
                desc: "Open a terminal and run the command for your distribution:",
                codes: [
                    "# Debian / Ubuntu\nsudo apt-get install libusb-1.0-0",
                    "# Fedora / RHEL\nsudo dnf install libusb1",
                    "# Arch\nsudo pacman -S libusb",
                ],
            },
            {
                title: "Assign permissions to your user",
                desc: "Open a terminal and run the command for your distribution:",
                codes: [
                    "# Debian / Ubuntu\nsudo usermod -aG lp,plugdev $USER",
                    "# Fedora / RHEL\nsudo usermod -aG lp,dialout $USER",
                    "# Arch\nsudo usermod -aG lp,uucp $USER",
                ],
            },
            {
                title: "Reconnect your printer and apply changes",
                desc: `Unplug and reconnect your printer. Then restart your system to apply the new group permissions.`,
            },
        ],
        OFFICE: CUPS_STEPS,
    };
}
