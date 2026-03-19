import tkinter as tk
from tkinter import ttk, messagebox
import json
import serial
import serial.tools.list_ports
import threading
import time
import os
import re 

# ---------------------------
# Global Settings and Files
# ---------------------------
LED_MAP_FILE = "led_map.json"

# Static mapping of IDs to location names.
id_to_name = {
    "101": "Van Cortlandt Park-242 St",
    "103": "238 St",
    "104": "231 St",
    "106": "Marble Hill-225 St",
    "107": "215 St",
    "108": "207 St",
    "109": "Dyckman St",
    "110": "191 St",
    "111": "181 St",
    "112": "168 St-Washington Hts",
    "113": "157 St",
    "114": "145 St",
    "115": "137 St-City College",
    "116": "125 St",
    "117": "116 St-Columbia University",
    "118": "Cathedral Pkwy (110 St)",
    "119": "103 St",
    "120": "96 St",
    "121": "86 St",
    "122": "79 St",
    "123": "72 St",
    "124": "66 St-Lincoln Center",
    "125": "59 St-Columbus Circle",
    "126": "50 St",
    "127": "Times Sq-42 St",
    "128": "34 St-Penn Station",
    "129": "28 St",
    "130": "23 St",
    "131": "18 St",
    "132": "14 St",
    "133": "Christopher St-Sheridan Sq",
    "134": "Houston St",
    "135": "Canal St",
    "136": "Franklin St",
    "137": "Chambers St",
    "138": "WTC Cortlandt",
    "139": "Rector St",
    "142": "South Ferry",
    "201": "Wakefield-241 St",
    "204": "Nereid Av",
    "205": "233 St",
    "206": "225 St",
    "207": "219 St",
    "208": "Gun Hill Rd",
    "209": "Burke Av",
    "210": "Allerton Av",
    "211": "Pelham Pkwy",
    "212": "Bronx Park East",
    "213": "E 180 St",
    "214": "West Farms Sq-E Tremont Av",
    "215": "174 St",
    "216": "Freeman St",
    "217": "Simpson St",
    "218": "Intervale Av",
    "219": "Prospect Av",
    "220": "Jackson Av",
    "221": "3 Av-149 St",
    "222": "149 St-Grand Concourse",
    "224": "135 St",
    "225": "125 St",
    "226": "116 St",
    "227": "Central Park North (110 St)",
    "228": "Park Place",
    "229": "Fulton St",
    "230": "Wall St",
    "231": "Clark St",
    "232": "Borough Hall",
    "233": "Hoyt St",
    "234": "Nevins St",
    "235": "Atlantic Av-Barclays Ctr",
    "236": "Bergen St",
    "237": "Grand Army Plaza",
    "238": "Eastern Pkwy-Brooklyn Museum",
    "239": "Franklin Av-Medgar Evers College",
    "241": "President St-Medgar Evers College",
    "242": "Sterling St",
    "243": "Winthrop St",
    "244": "Church Av",
    "245": "Beverly Rd",
    "246": "Newkirk Av-Little Haiti",
    "247": "Flatbush Av-Brooklyn College",
    "248": "Nostrand Av",
    "249": "Kingston Av",
    "250": "Crown Hts-Utica Av",
    "251": "Sutter Av-Rutland Rd",
    "252": "Saratoga Av",
    "253": "Rockaway Av",
    "254": "Junius St",
    "255": "Pennsylvania Av",
    "256": "Van Siclen Av",
    "257": "New Lots Av",
    "301": "Harlem-148 St",
    "302": "145 St",
    "401": "Woodlawn",
    "402": "Mosholu Pkwy",
    "405": "Bedford Park Blvd-Lehman College",
    "406": "Kingsbridge Rd",
    "407": "Fordham Rd",
    "408": "183 St",
    "409": "Burnside Av",
    "410": "176 St",
    "411": "Mt Eden Av",
    "412": "170 St",
    "413": "167 St",
    "414": "161 St-Yankee Stadium",
    "415": "149 St-Grand Concourse",
    "416": "138 St-Grand Concourse",
    "418": "Fulton St",
    "419": "Wall St",
    "420": "Bowling Green",
    "423": "Borough Hall",
    "501": "Eastchester-Dyre Av",
    "502": "Baychester Av",
    "503": "Gun Hill Rd",
    "504": "Pelham Pkwy",
    "505": "Morris Park",
    "601": "Pelham Bay Park",
    "602": "Buhre Av",
    "603": "Middletown Rd",
    "604": "Westchester Sq-E Tremont Av",
    "606": "Zerega Av",
    "607": "Castle Hill Av",
    "608": "Parkchester",
    "609": "St Lawrence Av",
    "610": "Morrison Av-Soundview",
    "611": "Elder Av",
    "612": "Whitlock Av",
    "613": "Hunts Point Av",
    "614": "Longwood Av",
    "615": "E 149 St",
    "616": "E 143 St-St Mary's St",
    "617": "Cypress Av",
    "618": "Brook Av",
    "619": "3 Av-138 St",
    "621": "125 St",
    "622": "116 St",
    "623": "110 St",
    "624": "103 St",
    "625": "96 St",
    "626": "86 St",
    "627": "77 St",
    "628": "68 St-Hunter College",
    "629": "59 St",
    "630": "51 St",
    "631": "Grand Central-42 St",
    "632": "33 St",
    "633": "28 St",
    "634": "23 St",
    "635": "14 St-Union Sq",
    "636": "Astor Pl",
    "637": "Bleecker St",
    "638": "Spring St",
    "639": "Canal St",
    "640": "Brooklyn Bridge-City Hall",
    "701": "Flushing-Main St",
    "702": "Mets-Willets Point",
    "705": "111 St",
    "706": "103 St-Corona Plaza",
    "707": "Junction Blvd",
    "708": "90 St-Elmhurst Av",
    "709": "82 St-Jackson Hts",
    "710": "74 St-Broadway",
    "711": "69 St",
    "712": "61 St-Woodside",
    "713": "52 St",
    "714": "46 St-Bliss St",
    "715": "40 St-Lowery St",
    "716": "33 St-Rawson St",
    "718": "Queensboro Plaza",
    "719": "Court Sq",
    "720": "Hunters Point Av",
    "721": "Vernon Blvd-Jackson Av",
    "723": "Grand Central-42 St",
    "724": "5 Av",
    "725": "Times Sq-42 St",
    "726": "34 St-Hudson Yards",
    "M18": "Delancey St-Essex St",
    "M19": "Bowery",
    "M20": "Canal St",
    "M21": "Chambers St",
    "M22": "Fulton St",
    "M23": "Broad St",
    "J27": "Broadway Junction",
    "J24": "Alabama Av",
    "J23": "Van Siclen Av",
    "J22": "Cleveland St",
    "J21": "Norwood Av",
    "J20": "Crescent St",
    "J19": "Cypress Hills",
    "J17": "75 St-Elderts Ln",
    "J16": "85 St-Forest Pkwy",
    "J15": "Woodhaven Blvd",
    "J14": "104 St",
    "J13": "111 St",
    "J12": "121 St",
    "G06": "Sutphin Blvd-Archer Av-JFK Airport",
    "G05": "Jamaica Center-Parsons/Archer",
    "J28": "Chauncey St",
    "J29": "Halsey St",
    "J30": "Gates Av",
    "J31": "Kosciuszko St",
    "M11": "Myrtle Av",
    "M12": "Flushing Av",
    "M13": "Lorimer St",
    "M14": "Hewes St",
    "M16": "Marcy Av",
    "L22": "Broadway Junction",
    "L24": "Atlantic Av",
    "L25": "Sutter Av",
    "L26": "Livonia Av",
    "L27": "New Lots Av",
    "L28": "East 105 St",
    "L08": "Bedford Av",
    "L06": "1 Av",
    "L05": "3 Av",
    "L03": "14 St-Union Sq",
    "L02": "6 Av",
    "L15": "Jefferson St",
    "L16": "DeKalb Av",
    "L17": "Myrtle-Wyckoff Avs",
    "L19": "Halsey St",
    "L20": "Wilson Av",
    "L21": "Bushwick Av-Aberdeen St",
    "L14": "Morgan Av",
    "L13": "Montrose Av",
    "L12": "Grand St",
    "L11": "Graham Av",
    "L10": "Lorimer St",
    "L29": "Canarsie-Rockaway Pkwy",
    "L01": "8 Av",
    "F27": "Church Av",
    "G31": "Flushing Av",
    "G30": "Broadway",
    "G29": "Metropolitan Av",
    "G28": "Nassau Av",
    "G26": "Greenpoint Av",
    "G24": "21 St",
    "G22": "Court Sq",
    "F21": "Carroll St",
    "F22": "Smith-9 Sts",
    "F23": "4 Av-9 St",
    "F24": "7 Av",
    "F25": "15 St-Prospect Park",
    "F26": "Fort Hamilton Pkwy",
    "G33": "Bedford-Nostrand Avs",
    "G34": "Classon Av",
    "G35": "Clinton-Washington Avs",
    "G36": "Fulton St",
    "A42": "Hoyt-Schermerhorn Sts",
    "F20": "Bergen St",
    "G32": "Myrtle-Willoughby Avs",
    "H02": "Aqueduct-N Conduit Av",
    "H03": "Howard Beach-JFK Airport",
    "A09": "168 St",
    "A57": "Grant Av",
    "A59": "80 St",
    "A60": "88 St",
    "A61": "Rockaway Blvd",
    "A63": "104 St",
    "A64": "111 St",
    "A65": "Ozone Park-Lefferts Blvd",
    "A12": "145 St",
    "A11": "155 St",
    "A10": "163 St-Amsterdam Av",
    "A47": "Kingston-Throop Avs",
    "A48": "Utica Av",
    "A49": "Ralph Av",
    "A50": "Rockaway Av",
    "A51": "Broadway Junction",
    "A52": "Liberty Av",
    "A53": "Van Siclen Av",
    "A54": "Shepherd Av",
    "A55": "Euclid Av",
    "A18": "103 St",
    "A17": "Cathedral Pkwy (110 St)",
    "A16": "116 St",
    "A15": "125 St",
    "A14": "135 St",
    "A40": "High St",
    "A41": "Jay St-MetroTech",
    "A43": "Lafayette Av",
    "A44": "Clinton-Washington Avs",
    "A45": "Franklin Av",
    "A46": "Nostrand Av",
    "A33": "Spring St",
    "A32": "W 4 St-Wash Sq",
    "A31": "14 St",
    "A30": "23 St",
    "A28": "34 St-Penn Station",
    "A27": "42 St-Port Authority Bus Terminal",
    "A25": "50 St",
    "A24": "59 St-Columbus Circle",
    "A22": "72 St",
    "A21": "81 St-Museum of Natural History",
    "A20": "86 St",
    "A19": "96 St",
    "A38": "Fulton St",
    "A36": "Chambers St",
    "A34": "Canal St",
    "H01": "Aqueduct Racetrack",
    "E01": "World Trade Center",
    "G09": "67 Av",
    "G08": "Forest Hills-71 Av",
    "F07": "75 Av",
    "F06": "Kew Gardens-Union Tpke",
    "F05": "Briarwood",
    "G07": "Jamaica-Van Wyck",
    "G14": "Jackson Hts-Roosevelt Av",
    "G13": "Elmhurst Av",
    "G12": "Grand Av-Newtown",
    "G11": "Woodhaven Blvd",
    "G10": "63 Dr-Rego Park",
    "F12": "5 Av/53 St",
    "D14": "7 Av",
    "G21": "Queens Plaza",
    "G20": "36 St",
    "G19": "Steinway St",
    "G18": "46 St",
    "G16": "Northern Blvd",
    "G15": "65 St",
    "F09": "Court Sq-23 St",
    "F11": "Lexington Av/53 St",
    "H14": "Beach 105 St",
    "H15": "Rockaway Park-Beach 116 St",
    "H12": "Beach 90 St",
    "H06": "Beach 67 St",
    "H07": "Beach 60 St",
    "H08": "Beach 44 St",
    "H09": "Beach 36 St",
    "H10": "Beach 25 St",
    "H11": "Far Rockaway-Mott Av",
    "H13": "Beach 98 St",
    "S04": "Botanic Garden",
    "S03": "Park Pl",
    "S01": "Franklin Av",
    "D26": "Prospect Park",
    "B23": "Bay 50 St",
    "D43": "Coney Island-Stillwell Av",
    "D03": "Bedford Park Blvd",
    "D01": "Norwood-205 St",
    "B13": "Fort Hamilton Pkwy",
    "B14": "50 St",
    "B15": "55 St",
    "B16": "62 St",
    "B17": "71 St",
    "B18": "79 St",
    "B19": "18 Av",
    "B20": "20 Av",
    "B21": "Bay Pkwy",
    "B22": "25 Av",
    "D17": "34 St-Herald Sq",
    "B12": "9 Av",
    "D12": "155 St",
    "D13": "145 St",
    "D15": "47-50 Sts-Rockefeller Ctr",
    "D16": "42 St-Bryant Pk",
    "D04": "Kingsbridge Rd",
    "D05": "Fordham Rd",
    "D06": "182-183 Sts",
    "D07": "Tremont Av",
    "D08": "174-175 Sts",
    "D09": "170 St",
    "D10": "167 St",
    "D11": "161 St-Yankee Stadium",
    "F01": "Jamaica-179 St",
    "F04": "Sutphin Blvd",
    "F03": "Parsons Blvd",
    "F02": "169 St",
    "D42": "W 8 St-NY Aquarium",
    "F34": "Avenue P",
    "F35": "Kings Hwy",
    "F36": "Avenue U",
    "F38": "Avenue X",
    "F39": "Neptune Av",
    "F29": "Ditmas Av",
    "F30": "18 Av",
    "F31": "Avenue I",
    "F32": "Bay Pkwy",
    "F33": "Avenue N",
    "B04": "21 St-Queensbridge",
    "B08": "Lexington Av/63 St",
    "B06": "Roosevelt Island",
    "D20": "W 4 St-Wash Sq",
    "D21": "Broadway-Lafayette St",
    "F14": "2 Av",
    "F15": "Delancey St-Essex St",
    "F16": "East Broadway",
    "F18": "York St",
    "B10": "57 St",
    "D18": "23 St",
    "D19": "14 St",
    "M04": "Fresh Pond Rd",
    "M01": "Middle Village-Metropolitan Av",
    "M08": "Myrtle-Wyckoff Avs",
    "M09": "Knickerbocker Av",
    "M10": "Central Av",
    "M06": "Seneca Av",
    "M05": "Forest Av",
    "R01": "Astoria-Ditmars Blvd",
    "N02": "8 Av",
    "N03": "Fort Hamilton Pkwy",
    "N04": "New Utrecht Av",
    "N05": "18 Av",
    "N06": "20 Av",
    "N07": "Bay Pkwy",
    "N08": "Kings Hwy",
    "N09": "Avenue U",
    "N10": "86 St",
    "R08": "39 Av-Dutch Kills",
    "R06": "36 Av",
    "R05": "Broadway",
    "R04": "30 Av",
    "R03": "Astoria Blvd",
    "R16": "Times Sq-42 St",
    "R15": "49 St",
    "R14": "57 St-7 Av",
    "R13": "5 Av/59 St",
    "R11": "Lexington Av/59 St",
    "R09": "Queensboro Plaza",
    "R27": "Whitehall St-South Ferry",
    "R24": "City Hall",
    "R25": "Cortlandt St",
    "R26": "Rector St",
    "R28": "Court St",
    "R29": "Jay St-MetroTech",
    "R30": "DeKalb Av",
    "R31": "Atlantic Av-Barclays Ctr",
    "R23": "Canal St",
    "R22": "Prince St",
    "R21": "8 St-NYU",
    "R20": "14 St-Union Sq",
    "R19": "23 St",
    "R18": "28 St",
    "R17": "34 St-Herald Sq",
    "R41": "59 St",
    "Q03": "72 St",
    "Q04": "86 St",
    "Q05": "96 St",
    "D29": "Beverley Rd",
    "D30": "Cortelyou Rd",
    "D31": "Newkirk Plaza",
    "D32": "Avenue H",
    "D33": "Avenue J",
    "D34": "Avenue M",
    "D35": "Kings Hwy",
    "D37": "Avenue U",
    "D38": "Neck Rd",
    "D39": "Sheepshead Bay",
    "D40": "Brighton Beach",
    "D41": "Ocean Pkwy",
    "D27": "Parkside Av",
    "D28": "Church Av",
    "D24": "Atlantic Av-Barclays Ctr",
    "Q01": "Canal St",
    "D25": "7 Av"
}

# Load live LED map (keys: "strip,LED", values: ID string)
if os.path.exists(LED_MAP_FILE):
    with open(LED_MAP_FILE, "r") as f:
        led_map = json.load(f)
else:
    led_map = {}

# ---------------------------
# Serial Communication Globals
# ---------------------------
ser = None             # Serial connection
serial_thread = None   # Background thread for reading serial data
stop_serial_thread = False

# Initialize default current LED position to 0,0.
current_strip = 0
current_pixel = 0

def send_serial(command: str):
    """Send a command over the serial port if connected."""
    global ser
    if ser and ser.is_open:
        try:
            ser.write(command.encode("utf-8"))
        except Exception as e:
            print(f"Error sending command: {e}")

def serial_reader():
    """Background thread: read serial lines and update current position.
    
    This function uses a regex to parse lines containing:
      "Strip: <number>, LED: <number>"
    """
    global ser, current_strip, current_pixel, stop_serial_thread
    # Regular expression to match "Strip: <num>, LED: <num>"
    pattern = re.compile(r"Strip:\s*(\d+),\s*LED:\s*(\d+)")
    while not stop_serial_thread:
        try:
            if ser and ser.in_waiting:
                line = ser.readline().decode("utf-8").strip()
                match = pattern.search(line)
                if match:
                    cs = int(match.group(1))
                    cp = int(match.group(2))
                    current_strip = cs
                    current_pixel = cp
                    app.root.after(0, app.update_current_position, cs, cp)
                print("Serial:", line)
            else:
                time.sleep(0.1)
        except Exception as e:
            print("Serial reader error:", e)
            time.sleep(0.5)

# ---------------------------
# Tkinter Application
# ---------------------------
class App:
    def __init__(self, root):
        self.root = root
        self.root.title("LED Navigation and Labeling")
        self.create_widgets()

    def create_widgets(self):
        # Serial Connection Frame
        serial_frame = ttk.LabelFrame(self.root, text="Serial Connection")
        serial_frame.pack(fill="x", padx=10, pady=5)
        
        ttk.Label(serial_frame, text="Select Serial Port:").pack(side="left", padx=5)
        self.port_var = tk.StringVar()
        self.port_combo = ttk.Combobox(serial_frame, textvariable=self.port_var, width=30)
        self.port_combo["values"] = self.get_serial_ports()
        self.port_combo.pack(side="left", padx=5)
        
        self.connect_btn = ttk.Button(serial_frame, text="Connect", command=self.connect_serial)
        self.connect_btn.pack(side="left", padx=5)
        
        self.status_label = ttk.Label(serial_frame, text="Disconnected", foreground="red")
        self.status_label.pack(side="left", padx=10)
        
        # LED Navigation Frame (includes manual position setting)
        nav_frame = ttk.LabelFrame(self.root, text="LED Navigation")
        nav_frame.pack(fill="x", padx=10, pady=5)
        
        self.current_pos_label = ttk.Label(nav_frame, text="Current Position: Strip 0, LED 0")
        self.current_pos_label.pack(padx=5, pady=5)
        
        nav_btn_frame = ttk.Frame(nav_frame)
        nav_btn_frame.pack(padx=5, pady=5)
        self.prev_btn = ttk.Button(nav_btn_frame, text="Previous", command=self.send_prev)
        self.prev_btn.pack(side="left", padx=5)
        self.next_btn = ttk.Button(nav_btn_frame, text="Next", command=self.send_next)
        self.next_btn.pack(side="left", padx=5)
        
        # Manual Position Setting (inside navigation frame)
        manual_inner = ttk.Frame(nav_frame)
        manual_inner.pack(padx=5, pady=5)
        ttk.Label(manual_inner, text="Set Position - Strip:").pack(side="left", padx=5)
        self.manual_strip_var = tk.StringVar(value="0")
        self.manual_strip_entry = ttk.Entry(manual_inner, textvariable=self.manual_strip_var, width=5)
        self.manual_strip_entry.pack(side="left", padx=5)
        ttk.Label(manual_inner, text="LED:").pack(side="left", padx=5)
        self.manual_led_var = tk.StringVar(value="0")
        self.manual_led_entry = ttk.Entry(manual_inner, textvariable=self.manual_led_var, width=5)
        self.manual_led_entry.pack(side="left", padx=5)
        self.set_pos_btn = ttk.Button(manual_inner, text="Set Position", command=self.set_position)
        self.set_pos_btn.pack(side="left", padx=5)
        
        # Label Current LED Frame
        label_frame = ttk.LabelFrame(self.root, text="Label Current LED")
        label_frame.pack(fill="x", padx=10, pady=5)
        ttk.Label(label_frame, text="Search Location:").pack(side="left", padx=5)
        self.search_var = tk.StringVar()
        self.search_entry = ttk.Entry(label_frame, textvariable=self.search_var, width=30)
        self.search_entry.pack(side="left", padx=5)
        self.search_entry.bind("<KeyRelease>", self.on_keyrelease)
        
        self.suggestion_var = tk.StringVar()
        self.suggestion_combo = ttk.Combobox(label_frame, textvariable=self.suggestion_var, width=40)
        self.suggestion_combo.pack(side="left", padx=5)
        self.all_options = [f"{k}: {v}" for k, v in id_to_name.items()]
        self.suggestion_combo["values"] = self.all_options
        
        self.label_btn = ttk.Button(label_frame, text="Submit", command=self.label_current)
        self.label_btn.pack(side="left", padx=5)
        self.label_btn.configure(takefocus=True)
        # Bind Enter key to trigger the label_current function.
        self.label_btn.bind("<Return>", lambda event: self.label_current())

        ttk.Style()
        self.label_btn.pack(side="left", padx=5)
        
        # Visited LED Map Frame
        map_frame = ttk.LabelFrame(self.root, text="Visited LED Map")
        map_frame.pack(fill="both", expand=True, padx=10, pady=5)
        self.map_text = tk.Text(map_frame, height=10)
        self.map_text.pack(fill="both", expand=True, padx=5, pady=5)
        self.update_map_display()

    def get_serial_ports(self):
        ports = serial.tools.list_ports.comports()
        return [port.device for port in ports]

    def connect_serial(self):
        global ser, serial_thread, stop_serial_thread
        port = self.port_var.get()
        if not port:
            messagebox.showerror("Error", "Please select a serial port")
            return
        try:
            ser = serial.Serial(port, 9600, timeout=1)
            self.status_label.config(text="Connected", foreground="green")
            messagebox.showinfo("Connected", f"Connected to {port}")
            send_serial("off\n")

            time.sleep(1)
            # Send the LED map to the microcontroller.
            for key, led_id in led_map.items():
                try:
                    strip, pixel = key.split(",")
                    print(f"going to {strip} at {pixel}")
                    time.sleep(0.2)
                    send_serial( f"jump {strip} {pixel}\n")
                except Exception as ex:
                    print("Error sending LED map:", ex)
            # Send a command to jump to the next unset LED.
            stop_serial_thread = False
            serial_thread = threading.Thread(target=serial_reader, daemon=True)
            serial_thread.start()
        except Exception as e:
            messagebox.showerror("Connection Error", f"Could not open serial port: {e}")
            self.status_label.config(text="Disconnected", foreground="red")

    def update_current_position(self, strip, pixel):
        self.current_pos_label.config(text=f"Current Position: Strip {strip}, LED {pixel}")

    def on_keyrelease(self, event):
        text = self.search_var.get().lower()
        suggestions = [option for option in self.all_options if text in option.lower()]
        self.suggestion_combo["values"] = suggestions
        if suggestions:
            self.suggestion_combo.current(0)

    def label_current(self):
        global current_strip, current_pixel, led_map
        if current_strip is None or current_pixel is None:
            messagebox.showerror("Error", "Current LED not set yet.")
            return
        selection = self.suggestion_var.get().strip()
        if not selection:
            messagebox.showerror("Error", "Please select a valid location from the drop down.")
            return
        try:
            led_id = selection.split(":")[0].strip()
        except Exception:
            messagebox.showerror("Error", "Invalid selection format.")
            return
        key = f"{current_strip},{current_pixel}"
        led_map[key] = led_id
        self.save_led_map()
        self.update_map_display()
        self.send_next()

    def set_position(self):
        global current_strip, current_pixel
        try:
            strip_val = int(self.manual_strip_var.get())
            led_val = int(self.manual_led_var.get())
        except ValueError:
            messagebox.showerror("Error", "Please enter valid integers for strip and LED.")
            return
        current_strip = strip_val
        current_pixel = led_val
        self.update_current_position(current_strip, current_pixel)
        cmd = f"jump {current_strip} {current_pixel}\n"
        send_serial(cmd)
        # No popup is shown on manual set per requirements.

    def send_next(self):
        send_serial("]\n")

    def send_prev(self):
        send_serial("[\n")

    def update_map_display(self):
        self.map_text.delete("1.0", tk.END)
        for key, led_id in reversed(led_map.items()):
            self.map_text.insert(tk.END, f"{key}: {led_id}\n")

    def save_led_map(self):
        with open(LED_MAP_FILE, "w") as f:
            json.dump(led_map, f, indent=2)

# ---------------------------
# Run the Application
# ---------------------------
if __name__ == "__main__":
    root = tk.Tk()
    app = App(root)
    try:
        root.mainloop()
    finally:
        stop_serial_thread = True
        if ser and ser.is_open:
            ser.close()
