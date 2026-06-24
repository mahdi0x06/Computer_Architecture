module ALU(
    input wire clk,
    input wire [1:0] ALUOp,
    input wire [31:0] Reg1, Reg2,
    output reg [31:0] Result,
    output wire zero,
    output wire ready
);
    wire [31:0] mul_res, div_res;
    wire mul_ready, div_ready;
    reg prev_mul, prev_div;

    initial begin
        prev_mul = 1'b0;
        prev_div = 1'b0;
    end

    always @(posedge clk) begin
        prev_mul <= (ALUOp == 2'b10);
        prev_div <= (ALUOp == 2'b11);
    end

    wire start_mul = (ALUOp == 2'b10) && !prev_mul;
    wire start_div = (ALUOp == 2'b11) && !prev_div;

    MUL mul(clk, start_mul, Reg1, Reg2, mul_res, mul_ready);
    DIV div(clk, start_div, Reg1, Reg2, div_res, div_ready);

    always @(*) begin
        case (ALUOp)
            2'b00: Result = Reg1 + Reg2;
            2'b01: Result = Reg1 - Reg2;
            2'b10: Result = mul_res;
            2'b11: Result = div_res;
            default: Result = 32'b0;
        endcase
    end
    
    assign zero = (Result == 32'b0) ? 1'b1 : 1'b0;
    assign ready = (ALUOp == 2'b10) ? mul_ready :
                   (ALUOp == 2'b11) ? div_ready : 1'b1;
endmodule

module MUL(
    input wire clk,
    input wire start,
    input wire [31:0] Reg1, Reg2,
    output reg [31:0] Result,
    output wire ready
);
    reg [5:0] count;
    reg [31:0] a, b;
    reg busy;
    
    initial busy = 1'b0;
    
    assign ready = ~busy & ~start;

    always @(posedge clk) begin
        if (start) begin
            a <= Reg1;
            b <= Reg2;
            Result <= 32'b0;
            count <= 6'b0;
            busy <= 1'b1;
        end else if (busy) begin
            if (a[0]) Result <= Result + b;
            a <= a >> 1;
            b <= b << 1;
            count <= count + 1;
            if (count == 31) busy <= 1'b0;
        end
    end
endmodule

module DIV(
    input wire clk,
    input wire start,
    input wire [31:0] Reg1, Reg2,
    output reg [31:0] Result,
    output wire ready
);
    reg [5:0] count;
    reg [31:0] q, r, b;
    reg busy;

    initial busy = 1'b0;

    assign ready = ~busy & ~start;

    always @(posedge clk) begin
        if (start) begin
            q <= Reg1;
            b <= Reg2;
            r <= 32'b0;
            Result <= 32'b0;
            count <= 32;
            busy <= 1'b1;
        end else if (busy) begin
            if ({r[30:0], q[31]} >= b) begin
                r <= {r[30:0], q[31]} - b;
                Result[count - 1] <= 1'b1;
            end else begin
                r <= {r[30:0], q[31]};
                Result[count - 1] <= 1'b0;
            end
            q <= q << 1;
            count <= count - 1;
            if (count == 1) busy <= 1'b0;
        end
    end
endmodule

module CU(
    input wire [5:0] opcode, funct,
    output reg [1:0] RegDst, MemtoReg, ALUOp,
    output reg RegWrite, ALUSrc, MemRead, MemWrite, Branch, Jump, Jal, Jr
);
    always @(*) begin
        RegDst = 2'b00;
        MemtoReg = 2'b00;
        ALUOp = 2'b00;
        RegWrite = 1'b0;
        ALUSrc = 1'b0;
        MemRead = 1'b0;
        MemWrite = 1'b0;
        Branch = 1'b0;
        Jump = 1'b0;
        Jal = 1'b0;
        Jr = 1'b0;

        case(opcode) 
            6'b000000:
                //JR
                if(funct == 6'b001000) Jr = 1'b1;
                else begin
                    RegWrite = 1'b1;
                    case(funct)
                        //ADD
                        6'b100000: ALUOp = 2'b00;
                        //SUB
                        6'b100010: ALUOp = 2'b01;
                        //MUL
                        6'b011000: ALUOp = 2'b10;
                        //DIV
                        6'b011010: ALUOp = 2'b11;
                        default: ; 
                    endcase
                end
            
            //LW
            6'b100011: begin ALUSrc = 1'b1; MemtoReg = 2'b01; MemRead = 1'b1; ALUOp = 2'b00; RegWrite = 1'b1; RegDst = 2'b01; end
            // SW
            6'b101011: begin ALUSrc = 1'b1; MemWrite = 1'b1; ALUOp = 2'b00; RegDst = 2'b01; end
            // ADDI
            6'b001000: begin ALUSrc = 1'b1; ALUOp = 2'b00; RegWrite = 1'b1; RegDst = 2'b01; end
            // SUBI
            6'b001001: begin ALUSrc = 1'b1; ALUOp = 2'b01; RegWrite = 1'b1; RegDst = 2'b01; end
            // LUI
            6'b001111: begin ALUSrc = 1'b1; RegWrite = 1'b1; RegDst = 2'b01; MemtoReg = 2'b11; end
            // J
            6'b000010: begin Jump = 1'b1; end
            // JAL
            6'b000011: begin MemtoReg = 2'b10; Jump = 1'b1; Jal = 1'b1; RegWrite = 1'b1; RegDst = 2'b10; end
            // BEQ
            6'b000100: begin Branch = 1'b1; ALUOp = 2'b01; end

            default: ; 
        endcase
    end
endmodule

module RF(
    input wire Clk, Rst, RegWrite,   
    input wire [4:0] ReadReg1, ReadReg2, WriteReg,
    input wire [31:0] Data,
    output wire [31:0] ReadData1, ReadData2,
    output wire [31:0] R0, R1, R2, R3, R4, R5, R6,
    R7, R8, R9, R10, R11, R12, R13, R14, R15,
    R16, R17, R18, R19, R20, R21, R22, R23, 
    R24, R25, R26, R27, R28, R29, R30, R31
);
    reg [31:0] Registers [0:31]; 
    integer i;

    initial begin
        for (i = 0; i < 32; i = i + 1) begin
            Registers[i] = 32'b0;
        end
    end

    assign ReadData1 = (ReadReg1 == 5'd0) ? 32'b0 : Registers[ReadReg1];
    assign ReadData2 = (ReadReg2 == 5'd0) ? 32'b0 : Registers[ReadReg2];
    
    assign {R31, R30, R29, R28, R27, R26, R25, R24, R23, R22, R21, R20, R19, R18, R17, R16, 
        R15, R14, R13, R12, R11, R10, R9, R8, R7, R6, R5, R4, R3, R2, R1, R0} = 
       {Registers[31], Registers[30], Registers[29], Registers[28], Registers[27],
        Registers[26], Registers[25], Registers[24], Registers[23], Registers[22],
        Registers[21], Registers[20], Registers[19], Registers[18], Registers[17],
        Registers[16], Registers[15], Registers[14], Registers[13], Registers[12],
        Registers[11], Registers[10], Registers[9], Registers[8], Registers[7], 
        Registers[6], Registers[5], Registers[4], Registers[3], Registers[2], 
        Registers[1], Registers[0]};

    always @(posedge Clk or posedge Rst) begin
        if (Rst == 1'b1) begin
            for (i = 0; i < 32; i = i + 1) begin
                Registers[i] <= 32'b0;
            end
        end else if (RegWrite == 1'b1 && WriteReg != 5'b00000) begin 
            Registers[WriteReg] <= Data;
        end
    end
endmodule

module main (
    input wire Clk,
    input wire Rst,
    input wire [31:0] InstrIn,
    input wire [31:0] DataIn,
    
    output wire [31:0] InstrAddr,
    output wire [31:0] DataAddr,
    output wire [31:0] DataOut,
    output wire DataWrite,
    
    output wire [31:0] R0, R1, R2, R3, R4, R5, R6, R7, R8, R9, R10, R11, R12, R13, R14, R15,
    output wire [31:0] R16, R17, R18, R19, R20, R21, R22, R23, R24, R25, R26, R27, R28, R29, R30, R31
);

    //PC
    reg [31:0] PC = 32'd2044; 
    wire [31:0] next_pc;
    wire [31:0] pc_plus_4 = PC + 32'd4;
    wire alu_ready;

    always @(posedge Clk or posedge Rst) begin
        if (Rst)
            PC <= 32'b0;
        else if (alu_ready)
            PC <= next_pc;
    end
    //

    assign InstrAddr = {2'b00, PC[31:2]};

    wire [5:0]  opcode = InstrIn[31:26];
    wire [4:0]  rs     = InstrIn[25:21];
    wire [4:0]  rt     = InstrIn[20:16];
    wire [4:0]  rd     = InstrIn[15:11];
    wire [5:0]  funct  = InstrIn[5:0];
    wire [15:0] imm    = InstrIn[15:0];
    wire [25:0] addr   = InstrIn[25:0];
    
    wire [31:0] sign_ext_imm = {{16{imm[15]}}, imm};

    //CU
    wire [1:0] RegDst, MemtoReg, ALUOp;
    wire RegWrite, ALUSrc, MemRead, MemWrite, Branch, Jump, Jal, Jr;

    CU control_unit (
        .opcode(opcode),
        .funct(funct),
        .RegDst(RegDst),
        .MemtoReg(MemtoReg),
        .ALUOp(ALUOp),
        .RegWrite(RegWrite),
        .ALUSrc(ALUSrc),
        .MemRead(MemRead),
        .MemWrite(MemWrite),
        .Branch(Branch),
        .Jump(Jump),
        .Jal(Jal),
        .Jr(Jr)
    );
    //

    //RF
    wire [4:0] write_reg;
    wire [31:0] write_data;
    wire [31:0] read_data1, read_data2;

    assign write_reg = (RegDst == 2'b10) ? 5'd31 :
                       (RegDst == 2'b01) ? rt :
                       rd;

    RF register_file (
        .Clk(Clk),
        .Rst(Rst),
        .RegWrite(RegWrite & alu_ready),
        .ReadReg1(rs),
        .ReadReg2(rt),
        .WriteReg(write_reg),
        .Data(write_data),
        .ReadData1(read_data1),
        .ReadData2(read_data2),
        .R0(R0), .R1(R1), .R2(R2), .R3(R3), .R4(R4), .R5(R5), .R6(R6), .R7(R7),
        .R8(R8), .R9(R9), .R10(R10), .R11(R11), .R12(R12), .R13(R13), .R14(R14), .R15(R15),
        .R16(R16), .R17(R17), .R18(R18), .R19(R19), .R20(R20), .R21(R21), .R22(R22), .R23(R23),
        .R24(R24), .R25(R25), .R26(R26), .R27(R27), .R28(R28), .R29(R29), .R30(R30), .R31(R31)
    );
    //

    //ALU
    wire [31:0] alu_in2;
    wire [31:0] alu_result;
    wire alu_zero;

    assign alu_in2 = ALUSrc ? sign_ext_imm : read_data2;

    ALU alu_inst (
        .clk(Clk),
        .ALUOp(ALUOp),
        .Reg1(read_data1),
        .Reg2(alu_in2),
        .Result(alu_result),
        .zero(alu_zero),
        .ready(alu_ready)
    );
    //

    wire is_memory_instruction = MemRead | MemWrite;
    assign DataAddr = is_memory_instruction ? {2'b00, alu_result[31:2]} : 32'h00004000;
    
    assign DataOut = read_data2;
    assign DataWrite = MemWrite & alu_ready;
    
    wire [31:0] lui_val = {imm, 16'b0};

    assign write_data = (MemtoReg == 2'b11) ? lui_val :
                        (MemtoReg == 2'b10) ? pc_plus_4 :
                        (MemtoReg == 2'b01) ? DataIn :
                        alu_result;

    wire [31:0] branch_target = pc_plus_4 + (sign_ext_imm << 2);
    wire [31:0] jump_target = {pc_plus_4[31:28], addr, 2'b00};
    wire do_branch = Branch & alu_zero;
    
    wire [31:0] pc_after_branch = do_branch ? branch_target : pc_plus_4;
    wire [31:0] pc_after_jump   = Jump ? jump_target : pc_after_branch;
    
    assign next_pc = Jr ? read_data1 : pc_after_jump;

endmodule