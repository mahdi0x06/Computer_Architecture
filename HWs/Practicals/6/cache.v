//سایز کش را یک کیلوبایت در نظر گرفتیم 
module L1_Cache (
    input Clk, Rst, CPU_Write, CPU_Read, MM_Done,
    input [31:0] CPU_DataIn, CPU_Addr,
    input [127:0] MM_DataIn,
    output CPU_Done, MM_Write, MM_Read,
    output [31:0] CPU_DataOut, MM_Addr,
    output [127:0] MM_DataOut 
);

    wire [3:0] offset;
    wire [5:0] ind;
    wire [21:0] tag;

    assign offset = CPU_Addr[3:0];
    assign ind = CPU_Addr[9:4];
    assign tag = CPU_Addr[31:10];

    reg [127:0] Data_arr [0:63];
    reg [21:0] tag_arr [0:63];
    reg valid_arr [0:63];

    wire current_valid = valid_arr[ind];
    wire [21:0] current_tag = tag_arr[ind];
    wire [127:0] current_data = Data_arr[ind];

    wire same_tag = (current_tag == tag);
    wire cache_hit = current_valid && same_tag;
    wire cache_miss = ~cache_hit;

    wire [31:0] word_0 = current_data[31:0];
    wire [31:0] word_1 = current_data[63:32];
    wire [31:0] word_2 = current_data[95:64];
    wire [31:0] word_3 = current_data[127:96];

    assign CPU_DataOut = (offset[3:2] == 2'b00) ? word_0 :
                         (offset[3:2] == 2'b01) ? word_1 :
                         (offset[3:2] == 2'b10) ? word_2 :
                         word_3;

    assign MM_Addr = {CPU_Addr[31:4], 4'b0000};
    assign MM_DataOut = (offset[3:2] == 2'b00) ? {current_data[127:32], CPU_DataIn} :
                        (offset[3:2] == 2'b01) ? {current_data[127:64], CPU_DataIn, current_data[31:0]} :
                        (offset[3:2] == 2'b10) ? {current_data[127:96], CPU_DataIn, current_data[63:0]} :
                        {CPU_DataIn, current_data[95:0]};

    reg state, next_state;
    integer i;

    always @(posedge Clk or posedge Rst) begin
        if(Rst == 1'b1) begin
            state <= 1'b0;
            for (i = 0;i < 64 ;i = i + 1) begin
                valid_arr[i] <= 1'b0;
            end
        end
        else 
            state <= next_state;    
    end
    

    always @(*) begin
        next_state = state;
        if(((CPU_Read && cache_miss) || (CPU_Write)) && ~state) 
            next_state = 1'b1;
        else if(MM_Done && state)
            next_state = 1'b0;
    end

    always @(posedge Clk) begin
        if(state && MM_Done) begin
            if(CPU_Read) begin
                valid_arr[ind] <= 1'b1;
                tag_arr[ind] <= tag;
                Data_arr[ind] <= MM_DataIn;
            end
            else if(CPU_Write && cache_hit) begin
                Data_arr[ind] <= MM_DataOut;
            end
        end
    end

    assign CPU_Done = (CPU_Read && cache_hit && ~state) || (state && MM_Done);
    assign MM_Read = (CPU_Read && cache_miss);
    assign MM_Write = CPU_Write;
    
endmodule